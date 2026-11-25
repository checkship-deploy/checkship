import type { EnvironmentVariablesPerService } from '#lib/resources/apps.ts';
import type { Template, TemplateCredential, TemplateField } from './types.ts';

/** Values the person typed or checkship generated, by variable name. */
export type TemplateValues = Record<string, string>;

/** Why a field blocks Deploy, as a translation key for FieldMessage. */
export type FieldProblem =
	'templates.field.required' | 'templates.field.too_short' | 'templates.field.email';

/** A random secret of `length` hex characters, from the browser's crypto source. */
export function generateSecret(
	length: number,
	fill: (bytes: Uint8Array<ArrayBuffer>) => Uint8Array = (b) => crypto.getRandomValues(b)
): string {
	const bytes = fill(new Uint8Array(Math.ceil(length / 2)));
	return Array.from(bytes, (b) => b.toString(16).padStart(2, '0'))
		.join('')
		.slice(0, length);
}

/** A generated field's value: a new secret, after the field's prefix if it has one. */
export function newSecret(field: TemplateField): string {
	return (field.prefix ?? '') + generateSecret(field.length ?? 32);
}

/** The secret a value carries, without its field's prefix: what people copy and connect with. */
export function secretOf(field: TemplateField, value: string | undefined): string {
	const v = value ?? '';
	return field.prefix && v.startsWith(field.prefix) ? v.slice(field.prefix.length) : v;
}

/** Starting values: generated secrets and optional switches' defaults. */
export function initialValues(template: Template): TemplateValues {
	const values: TemplateValues = {};
	for (const field of template.fields) {
		if (field.kind === 'generated') values[field.key] = newSecret(field);
		if (field.kind === 'optional') values[field.key] = String(field.default ?? false);
	}
	return values;
}

/** A derived field's value from the app URL, or undefined when there is no URL to derive from. */
export function derivedValue(field: TemplateField, url: string | undefined): string | undefined {
	if (field.kind !== 'derived' || !url) return undefined;
	return field.derive === 'url-slash' ? `${url}/` : url;
}

/**
 * Fields the person must type: required ones, and derived ones when the target has no URL to
 * derive them from (a manual proxy).
 */
export function inputFields(template: Template, url: string | undefined): TemplateField[] {
	return template.fields.filter((f) => f.kind === 'required' || (f.kind === 'derived' && !url));
}

/** What is wrong with a typed field, if anything. */
export function fieldProblem(
	field: TemplateField,
	value: string | undefined
): FieldProblem | undefined {
	const v = (value ?? '').trim();
	if (!v) return 'templates.field.required';
	if (field.minLength && v.length < field.minLength) return 'templates.field.too_short';
	if (field.email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v)) return 'templates.field.email';
	return undefined;
}

/** The typed fields that still block Deploy. */
export function blockingFields(
	template: Template,
	values: TemplateValues,
	url: string | undefined
): TemplateField[] {
	return inputFields(template, url).filter((f) => !!fieldProblem(f, values[f.key]));
}

/** Whether the template asks for anything before it can deploy (on a target with a URL). */
export function needsSetup(template: Template): boolean {
	return template.fields.some((f) => f.kind === 'required');
}

/**
 * The app variables to create the app with: every field's value for the template's service.
 * Derived values come from the URL (or what was typed when there is none). Empty values are left
 * out, so the compose file's own value or the image default applies.
 */
export function resolveVars(
	template: Template,
	values: TemplateValues,
	url: string | undefined
): EnvironmentVariablesPerService {
	const vars: Record<string, string> = {};
	for (const field of template.fields) {
		const value =
			field.kind === 'derived'
				? (derivedValue(field, url) ?? values[field.key]?.trim())
				: field.kind === 'required'
					? values[field.key]?.trim()
					: values[field.key];
		if (value) vars[field.key] = value;
	}
	return Object.keys(vars).length ? { [template.service]: vars } : {};
}

/**
 * A free application name: `base` when nobody uses it, else `base-2`, `base-3`… An app's name must
 * be unique on its target, and templates default to their id, so a second Gitea gets `gitea-2`.
 */
export function freeName(base: string, taken: string[]): string {
	if (!taken.includes(base)) return base;
	let n = 2;
	while (taken.includes(`${base}-${n}`)) n++;
	return `${base}-${n}`;
}

/**
 * Where the person's other applications on the target reach the template's service:
 * `<service>.<app>`, with -staging in staging. checkship gives every service with a published
 * port this alias on the target's network (`InternalHost` in config_snapshot.go).
 */
export function internalHost(
	template: Template,
	name: string,
	environment: 'production' | 'staging' = 'production'
): string {
	return `${template.service}.${name}${environment === 'staging' ? '-staging' : ''}`;
}

/**
 * The template's connection string for `host`, with every `{FIELD_KEY}` replaced by that field's
 * secret, or undefined when the template has none.
 */
export function connectionUrl(
	template: Template,
	values: TemplateValues,
	host: string
): string | undefined {
	const pattern = template.connect?.url;
	if (!pattern) return undefined;
	return pattern.replace(/\{([A-Za-z0-9_]+)\}/g, (match, key: string) => {
		if (key === 'host') return host;
		const field = template.fields.find((f) => f.key === key);
		return field ? secretOf(field, values[key]) : match;
	});
}

/**
 * Whether the template publishes a web port, so the app gets a page at its URL. A database or a
 * cache only publishes "/tcp" ports: its URL would open nothing.
 */
export function hasWebPage(template: Template): boolean {
	return /^\s+- "\d+:\d+"$/m.test(template.compose);
}

/**
 * A credential's text, with `{url}` and every `{FIELD_KEY}` filled from the values (the secret
 * alone, without the field's prefix). A placeholder without a value stays empty.
 */
export function credentialValue(
	template: Template,
	credential: TemplateCredential,
	values: TemplateValues,
	url: string | undefined
): string {
	return credential.value.replace(/\{([A-Za-z0-9_]+)\}/g, (_match, key: string) => {
		if (key === 'url') return url ?? '';
		const field = template.fields.find((f) => f.key === key);
		return field ? secretOf(field, values[key]) : (values[key] ?? '');
	});
}

/** The values a deployed app runs with: its variables for the template's service. */
export function valuesOf(
	template: Template,
	vars: EnvironmentVariablesPerService | undefined
): TemplateValues {
	return { ...(vars?.[template.service] ?? {}) };
}
