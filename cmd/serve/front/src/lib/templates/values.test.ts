import { describe, expect, test } from 'vitest';
import { appUrl } from '../app-url.ts';
import { templateById } from './catalog.ts';
import type { Template } from './types.ts';
import {
	blockingFields,
	connectionUrl,
	credentialValue,
	derivedValue,
	fieldProblem,
	freeName,
	generateSecret,
	hasWebPage,
	initialValues,
	inputFields,
	internalHost,
	needsSetup,
	newSecret,
	resolveVars,
	secretOf,
	strongSecret,
	valuesOf,
	type TemplateValues
} from './values.ts';

const get = (id: string): Template => templateById(id)!;
const url = 'http://gitea.docker.localhost';

describe('generateSecret', () => {
	test('returns the asked number of hex characters', () => {
		expect(generateSecret(48)).toMatch(/^[0-9a-f]{48}$/);
		expect(generateSecret(7)).toMatch(/^[0-9a-f]{7}$/);
	});

	test('uses the random source', () => {
		const zeros = generateSecret(8, (b) => b.fill(0));
		const ones = generateSecret(8, (b) => b.fill(255));
		expect(zeros).toBe('00000000');
		expect(ones).toBe('ffffffff');
	});

	test('differs between calls', () => {
		expect(generateSecret(32)).not.toBe(generateSecret(32));
	});
});

describe('initialValues', () => {
	test('generates secrets and sets optional defaults', () => {
		const values = initialValues(get('vaultwarden'));
		expect(values.ADMIN_TOKEN).toMatch(/^[0-9a-f]{48}$/);
		expect(values.SIGNUPS_ALLOWED).toBe('true');
		expect(values.DOMAIN).toBeUndefined();
	});

	test('leaves required fields empty', () => {
		expect(initialValues(get('gotify'))).toEqual({});
	});
});

describe('derived values', () => {
	test('follow the app URL', () => {
		const gitea = get('gitea').fields[0];
		const vaultwarden = get('vaultwarden').fields[0];
		expect(derivedValue(gitea, url)).toBe(`${url}/`);
		expect(derivedValue(vaultwarden, 'https://vaultwarden.example.com')).toBe(
			'https://vaultwarden.example.com'
		);
	});

	test('become inputs when the target has no URL', () => {
		expect(inputFields(get('gitea'), url)).toEqual([]);
		expect(inputFields(get('gitea'), undefined).map((f) => f.key)).toEqual([
			'GITEA__server__ROOT_URL'
		]);
	});
});

describe('required settings', () => {
	const gotify = get('gotify');
	const [username, password] = gotify.fields;

	test('mark the template as needing setup', () => {
		expect(needsSetup(gotify)).toBe(true);
		expect(needsSetup(get('uptime-kuma'))).toBe(false);
		expect(needsSetup(get('vaultwarden'))).toBe(false);
	});

	test('report what is wrong', () => {
		expect(fieldProblem(username, '')).toBe('templates.field.required');
		expect(fieldProblem(username, '   ')).toBe('templates.field.required');
		expect(fieldProblem(username, 'ops')).toBeUndefined();
		expect(fieldProblem(password, 'hunter2')).toBe('templates.field.too_short');
		expect(fieldProblem(password, 'correct-horse')).toBeUndefined();
	});

	test('block Deploy until both are set and the password is long enough', () => {
		expect(blockingFields(gotify, {}, url)).toHaveLength(2);
		expect(blockingFields(gotify, { GOTIFY_DEFAULTUSER_NAME: 'ops' }, url)).toHaveLength(1);
		expect(
			blockingFields(
				gotify,
				{ GOTIFY_DEFAULTUSER_NAME: 'ops', GOTIFY_DEFAULTUSER_PASS: 'short' },
				url
			)
		).toEqual([password]);
		expect(
			blockingFields(
				gotify,
				{ GOTIFY_DEFAULTUSER_NAME: 'ops', GOTIFY_DEFAULTUSER_PASS: 'long-enough' },
				url
			)
		).toEqual([]);
	});

	test('never block a template that asks for nothing', () => {
		expect(blockingFields(get('uptime-kuma'), {}, url)).toEqual([]);
		expect(blockingFields(get('vaultwarden'), initialValues(get('vaultwarden')), url)).toEqual([]);
	});
});

describe('resolveVars', () => {
	test('puts every value under the template service', () => {
		const vaultwarden = get('vaultwarden');
		const values: TemplateValues = { ...initialValues(vaultwarden), SIGNUPS_ALLOWED: 'false' };
		const vars = resolveVars(vaultwarden, values, 'http://vaultwarden.docker.localhost');
		expect(vars).toEqual({
			vaultwarden: {
				DOMAIN: 'http://vaultwarden.docker.localhost',
				ADMIN_TOKEN: values.ADMIN_TOKEN,
				SIGNUPS_ALLOWED: 'false'
			}
		});
	});

	test('derives each environment from its own address', () => {
		const target = 'http://docker.localhost:8090';
		const gitea = get('gitea');
		const of = (environment: 'production' | 'staging') =>
			resolveVars(gitea, {}, appUrl(target, 'gitea', environment)).gitea;
		expect(of('production')).toEqual({
			GITEA__server__ROOT_URL: 'http://gitea.docker.localhost:8090/'
		});
		expect(of('staging')).toEqual({
			GITEA__server__ROOT_URL: 'http://gitea-staging.docker.localhost:8090/'
		});
	});

	test('trims typed values', () => {
		const vars = resolveVars(
			get('gotify'),
			{ GOTIFY_DEFAULTUSER_NAME: ' ops ', GOTIFY_DEFAULTUSER_PASS: 'long-enough' },
			url
		);
		expect(vars.gotify).toEqual({
			GOTIFY_DEFAULTUSER_NAME: 'ops',
			GOTIFY_DEFAULTUSER_PASS: 'long-enough'
		});
	});

	test('uses a typed value for a derived field without a target URL', () => {
		const vars = resolveVars(
			get('gitea'),
			{ GITEA__server__ROOT_URL: 'https://git.example.com/' },
			undefined
		);
		expect(vars).toEqual({ gitea: { GITEA__server__ROOT_URL: 'https://git.example.com/' } });
	});

	test('is empty for a template without settings', () => {
		expect(resolveVars(get('uptime-kuma'), {}, url)).toEqual({});
	});
});

describe('freeName', () => {
	test('keeps the template id when it is free', () => {
		expect(freeName('gitea', ['gotify'])).toBe('gitea');
	});

	test('counts up past names already used on the target', () => {
		expect(freeName('gitea', ['gitea'])).toBe('gitea-2');
		expect(freeName('gitea', ['gitea', 'gitea-2', 'gitea-3'])).toBe('gitea-4');
	});
});

describe('prefixed secrets', () => {
	const valkey = get('valkey');
	const [password] = valkey.fields;

	test('carry their prefix in the variable and not in what people copy', () => {
		const value = newSecret(password);
		expect(value).toMatch(/^--requirepass [0-9a-f]{32}$/);
		expect(secretOf(password, value)).toMatch(/^[0-9a-f]{32}$/);
		expect(initialValues(get('mailpit')).MP_UI_AUTH).toMatch(/^admin:[0-9a-f]{24}$/);
	});

	test('leave a value without the prefix as it is', () => {
		expect(secretOf(password, 'plain')).toBe('plain');
		expect(secretOf(get('postgresql').fields[0], 'abc')).toBe('abc');
	});
});

describe('connecting from other apps', () => {
	test('uses the service alias, with -staging in staging', () => {
		expect(internalHost(get('postgresql'), 'db')).toBe('postgres.db');
		expect(internalHost(get('postgresql'), 'db', 'staging')).toBe('postgres.db-staging');
	});

	test('fills the connection string with the secret alone', () => {
		const values: TemplateValues = { VALKEY_EXTRA_FLAGS: '--requirepass s3cret' };
		expect(connectionUrl(get('valkey'), values, 'valkey.cache')).toBe(
			'redis://default:s3cret@valkey.cache:6379/0'
		);
		expect(connectionUrl(get('postgresql'), { POSTGRES_PASSWORD: 'pw' }, 'postgres.db')).toBe(
			'postgres://app:pw@postgres.db:5432/app'
		);
	});

	test('has nothing for a template other apps do not connect to', () => {
		expect(connectionUrl(get('gitea'), {}, 'gitea.git')).toBeUndefined();
	});
});

describe('hasWebPage', () => {
	test('is false for a database or cache that only publishes tcp ports', () => {
		expect(hasWebPage(get('postgresql'))).toBe(false);
		expect(hasWebPage(get('valkey'))).toBe(false);
	});

	test('is true when a web port is published beside tcp ones', () => {
		expect(hasWebPage(get('mailpit'))).toBe(true);
		expect(hasWebPage(get('rustfs'))).toBe(true);
		expect(hasWebPage(get('gitea'))).toBe(true);
	});
});

describe('an email field', () => {
	const pgadmin = get('pgadmin');
	const email = pgadmin.fields.find((f) => f.key === 'PGADMIN_DEFAULT_EMAIL')!;

	test('needs setup and checks the address before Deploy', () => {
		expect(needsSetup(pgadmin)).toBe(true);
		expect(fieldProblem(email, '')).toBe('templates.field.required');
		expect(fieldProblem(email, 'admin')).toBe('templates.field.email');
		expect(fieldProblem(email, 'you@example')).toBe('templates.field.email');
		expect(fieldProblem(email, 'you@example.com')).toBeUndefined();
		expect(blockingFields(pgadmin, initialValues(pgadmin), url)).toEqual([email]);
	});
});

describe('sign-in details', () => {
	test('come from the app variables, with prefixes left out', () => {
		const mailpit = get('mailpit');
		const values = valuesOf(mailpit, { mailpit: { MP_UI_AUTH: 'admin:s3cret' } });
		const [user, password] = mailpit.credentials!;
		expect(credentialValue(mailpit, user, values, undefined)).toBe('admin');
		expect(credentialValue(mailpit, password, values, undefined)).toBe('s3cret');
		expect(password.secret).toBe(true);
	});

	test('fill the app URL in links', () => {
		const vaultwarden = get('vaultwarden');
		const page = vaultwarden.credentials![0];
		expect(credentialValue(vaultwarden, page, {}, 'http://vw.docker.localhost:8090')).toBe(
			'http://vw.docker.localhost:8090/admin'
		);
	});

	test('show what was typed, such as Gotify’s admin', () => {
		const gotify = get('gotify');
		const values = valuesOf(gotify, {
			gotify: { GOTIFY_DEFAULTUSER_NAME: 'ops', GOTIFY_DEFAULTUSER_PASS: 'pw' }
		});
		expect(gotify.credentials!.map((c) => credentialValue(gotify, c, values, undefined))).toEqual([
			'ops',
			'pw'
		]);
	});

	test('are empty for a service with no variables yet', () => {
		expect(valuesOf(get('nats'), undefined)).toEqual({});
	});
});

describe('strongSecret', () => {
	test('mixes every class OpenObserve asks for, in URL-safe characters', () => {
		for (let i = 0; i < 50; i++) {
			const secret = strongSecret(24);
			expect(secret).toHaveLength(24);
			expect(secret).toMatch(/[a-z]/);
			expect(secret).toMatch(/[A-Z]/);
			expect(secret).toMatch(/[0-9]/);
			expect(secret).toMatch(/[-_.~]/);
			expect(secret).toMatch(/^[A-Za-z0-9\-_.~]+$/);
		}
	});

	test('is what OpenObserve’s password field generates', () => {
		const password = initialValues(get('openobserve')).ZO_ROOT_USER_PASSWORD;
		expect(password).toMatch(/[a-z]/);
		expect(password).toMatch(/[A-Z]/);
		expect(password).toMatch(/[-_.~]/);
	});
});
