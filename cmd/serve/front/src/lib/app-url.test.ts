import { describe, expect, test } from 'vitest';
import { appUrl, isSecureContextUrl } from './app-url.ts';

describe('appUrl', () => {
	test('builds the default service URL from the target', () => {
		expect(appUrl('http://docker.localhost', 'gotify')).toBe('http://gotify.docker.localhost');
		expect(appUrl('https://example.com:8443', 'memos')).toBe('https://memos.example.com:8443');
	});

	test('adds -staging in staging', () => {
		expect(appUrl('https://example.com', 'memos', 'staging')).toBe(
			'https://memos-staging.example.com'
		);
	});

	test('is undefined without a target URL or a name', () => {
		expect(appUrl(undefined, 'gotify')).toBeUndefined();
		expect(appUrl('not a url', 'gotify')).toBeUndefined();
		expect(appUrl('http://docker.localhost', '')).toBeUndefined();
	});
});

describe('isSecureContextUrl', () => {
	test('accepts https and localhost addresses', () => {
		expect(isSecureContextUrl('https://vault.example.com')).toBe(true);
		expect(isSecureContextUrl('http://vaultwarden.docker.localhost')).toBe(true);
		expect(isSecureContextUrl('http://localhost:8080')).toBe(true);
	});

	test('rejects other http addresses', () => {
		expect(isSecureContextUrl('http://vaultwarden.lab.internal')).toBe(false);
		expect(isSecureContextUrl(undefined)).toBe(false);
	});
});
