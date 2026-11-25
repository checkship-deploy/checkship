import { describe, expect, test } from 'vitest';
import type { DeploymentDetail } from '#lib/resources/deployments.ts';
import { liveUrl, socketAddresses } from './app-state.ts';

/** A deployment with the given services, the rest left out. */
const deployment = (services: unknown[]) =>
	({ state: { services } }) as unknown as DeploymentDetail;

const postgres = deployment([
	{
		name: 'postgres',
		image: 'postgres:18.7-alpine',
		entrypoints: [
			{
				name: 'tcp1',
				router: 'tcp',
				port: 5432,
				is_custom: true,
				url: 'tcp://docker.localhost:32768'
			}
		]
	}
]);

describe('socketAddresses', () => {
	test('gives a tcp service its address from outside and from other apps', () => {
		expect(socketAddresses(postgres, 'db', 'production')).toEqual([
			{ service: 'postgres', outside: 'tcp://docker.localhost:32768', inside: 'postgres.db:5432' }
		]);
		expect(socketAddresses(postgres, 'db', 'staging')[0].inside).toBe('postgres.db-staging:5432');
	});

	test('leaves web entrypoints out', () => {
		const web = deployment([
			{
				name: 'web',
				image: 'nginx',
				entrypoints: [
					{ name: 'http1', router: 'http', port: 80, is_custom: false, url: 'http://web.x' }
				]
			}
		]);
		expect(socketAddresses(web, 'site', 'production')).toEqual([]);
		expect(socketAddresses(undefined, 'site', 'production')).toEqual([]);
	});
});

describe('liveUrl', () => {
	test('never offers a tcp address to open in a browser', () => {
		expect(liveUrl(postgres)).toBeUndefined();
	});
});
