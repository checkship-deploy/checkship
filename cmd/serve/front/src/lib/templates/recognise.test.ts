import { describe, expect, test } from 'vitest';
import { templates } from './catalog.ts';
import { imageRepository, templateOfApp, templateOfSource } from './recognise.ts';

const raw = (data: string) => ({ discriminator: 'raw' as const, data });

describe('imageRepository', () => {
	test('drops the tag, the digest and the default registry', () => {
		expect(imageRepository('postgres:18.7-alpine')).toBe('postgres');
		expect(imageRepository('docker.io/library/postgres:17')).toBe('postgres');
		expect(imageRepository('"valkey/valkey:9.1.4-alpine"')).toBe('valkey/valkey');
		expect(imageRepository('getmeili/meilisearch@sha256:abc')).toBe('getmeili/meilisearch');
	});

	test('keeps a registry port, which is not a tag', () => {
		expect(imageRepository('localhost:5000/team/api')).toBe('localhost:5000/team/api');
		expect(imageRepository('docker.gitea.com/gitea:28.4.0')).toBe('docker.gitea.com/gitea');
	});
});

describe('templateOfSource', () => {
	test('recognises every template from its own compose file', () => {
		for (const template of templates) {
			expect(templateOfSource(raw(template.compose))?.id, template.id).toBe(template.id);
		}
	});

	test('recognises another tag of the same image', () => {
		const compose =
			'services:\n  db:\n    image: postgres:17\n    ports:\n      - "5432:5432/tcp"\n';
		expect(templateOfSource(raw(compose))?.id).toBe('postgresql');
	});

	test('names no template for an app with several images, a build or an unknown image', () => {
		const several =
			'services:\n  api:\n    image: team/api:1\n  db:\n    image: postgres:18.7-alpine\n';
		const built = 'services:\n  db:\n    image: postgres:18\n    build: .\n';
		const unknown = 'services:\n  web:\n    image: nginx:1.29\n';
		expect(templateOfSource(raw(several))).toBeUndefined();
		expect(templateOfSource(raw(built))).toBeUndefined();
		expect(templateOfSource(raw(unknown))).toBeUndefined();
	});

	test('names none for a git or archive source, which carries no compose file', () => {
		expect(templateOfSource({ discriminator: 'git', data: { branch: 'main', hash: 'a' } })).toBe(
			undefined
		);
		expect(templateOfSource({ discriminator: 'archive', data: 'app.tar.gz' })).toBeUndefined();
		expect(templateOfSource(undefined)).toBeUndefined();
	});
});

describe('templateOfApp', () => {
	const mailpit = templates.find((t) => t.id === 'mailpit')!;

	test('reads production, else staging', () => {
		expect(
			templateOfApp({ latest_deployments: { production: { source: raw(mailpit.compose) } } })?.id
		).toBe('mailpit');
		expect(
			templateOfApp({ latest_deployments: { staging: { source: raw(mailpit.compose) } } })?.id
		).toBe('mailpit');
		expect(templateOfApp({ latest_deployments: {} })).toBeUndefined();
	});
});
