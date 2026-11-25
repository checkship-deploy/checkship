import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, test } from 'vitest';
import en from '#lib/localization/en.ts';
import fr from '#lib/localization/fr.ts';
import { categories, templateById, templates } from './catalog.ts';

const staticDir = resolve(__dirname, '../../../static');
/** The docs site serves its own copies of the logos (docs/guide/templates.md). */
const docsLogos = resolve(__dirname, '../../../../../../docs/public/template-logos');

/** Top-level keys under `services:` in a compose file (two-space indented names). */
function serviceNames(compose: string): string[] {
	const lines = compose.split('\n');
	const start = lines.indexOf('services:');
	const names: string[] = [];
	for (const line of lines.slice(start + 1)) {
		if (/^\S/.test(line)) break;
		const match = /^ {2}([\w-]+):\s*$/.exec(line);
		if (match) names.push(match[1]);
	}
	return names;
}

describe('the template catalog', () => {
	test('has unique ids that are valid application names', () => {
		const ids = templates.map((t) => t.id);
		expect(new Set(ids).size).toBe(ids.length);
		for (const id of ids) {
			expect(id).toMatch(/^[a-z0-9-_]+$/);
			expect(id.endsWith('-staging')).toBe(false);
		}
	});

	test('uses known categories', () => {
		for (const t of templates) expect(categories).toContain(t.category);
	});

	test('finds a template by id', () => {
		expect(templateById('gotify')?.name).toBe('Gotify');
		expect(templateById('nope')).toBeUndefined();
	});

	describe.each(templates)('$name', (template) => {
		test('has one compose service, named after the template service', () => {
			expect(serviceNames(template.compose)).toEqual([template.service]);
		});

		test('pins the image it shows', () => {
			expect(template.compose).toContain(`image: ${template.image}`);
			expect(template.image).not.toMatch(/:latest$/);
		});

		test('can be deployed from a raw compose file', () => {
			// A raw source writes nothing but compose.yml: no build context.
			expect(template.compose).not.toMatch(/^\s+build:/m);
			// checkship ignores ports without a host part; http ones get the app URL, /tcp ones a
			// port of their own on the target.
			const ports = [...template.compose.matchAll(/^\s+- "(\d[^"]*)"$/gm)].map((m) => m[1]);
			expect(ports.length).toBeGreaterThan(0);
			for (const port of ports) expect(port).toMatch(/^\d+:\d+(\/tcp)?$/);
			// Only a service other apps connect to may have no web port.
			if (!template.connect) expect(ports.some((p) => !p.endsWith('/tcp'))).toBe(true);
			// Bind mounts are not kept between deployments: named volumes only.
			expect(template.compose).not.toMatch(/^\s+- \.{0,2}\//m);
			expect(template.compose).toMatch(/^volumes:/m);
			expect(template.compose).toContain('restart: unless-stopped');
		});

		test('ships its logo files and a notice entry', () => {
			const notice = readFileSync(resolve(staticDir, 'template-logos/NOTICE.md'), 'utf8');
			for (const logo of [template.logo, template.logoDark].filter((l): l is string => !!l)) {
				const file = resolve(staticDir, `.${logo}`);
				expect(existsSync(file), logo).toBe(true);
				expect(readFileSync(file, 'utf8')).toMatch(/<svg[\s>]/);
				expect(notice).toContain(logo.split('/').pop());
				const copy = resolve(docsLogos, logo.split('/').pop()!);
				expect(existsSync(copy), `docs copy of ${logo}`).toBe(true);
				expect(readFileSync(copy, 'utf8'), `docs copy of ${logo}`).toBe(readFileSync(file, 'utf8'));
			}
		});

		test('has its text in every language', () => {
			const keys = [
				template.description,
				template.after,
				...template.fields.flatMap((f) => [f.label, f.help].filter((k) => !!k))
			];
			for (const key of keys) {
				expect(en.translations).toHaveProperty([key as string]);
				expect(fr.translations).toHaveProperty([key as string]);
			}
		});

		test('publishes the port other apps connect to, with a string they can fill', () => {
			if (!template.connect) return;
			const { port, url } = template.connect;
			expect(template.compose).toMatch(new RegExp(`- "${port}:${port}(/tcp)?"`));
			for (const [, key] of (url ?? '').matchAll(/\{([A-Za-z0-9_]+)\}/g)) {
				expect(key === 'host' || template.fields.some((f) => f.key === key), key).toBe(true);
			}
		});

		test('reads its sign-in from its own fields, in every language', () => {
			for (const credential of template.credentials ?? []) {
				for (const [, key] of credential.value.matchAll(/\{([A-Za-z0-9_]+)\}/g)) {
					expect(key === 'url' || template.fields.some((f) => f.key === key), key).toBe(true);
				}
				expect(en.translations).toHaveProperty([credential.label as string]);
				expect(fr.translations).toHaveProperty([credential.label as string]);
			}
		});

		test('says how to sign in, or that the first visit creates the account', () => {
			// Apps you open without anything from checkship create their account in the app.
			const createdInApp = ['uptime-kuma', 'memos', 'gitea'];
			expect(!!template.credentials?.length || createdInApp.includes(template.id)).toBe(true);
		});

		test('has distinct field names', () => {
			const keys = template.fields.map((f) => f.key);
			expect(new Set(keys).size).toBe(keys.length);
		});
	});
});
