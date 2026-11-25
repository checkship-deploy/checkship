import type { Environment } from '#lib/resources/deployments.ts';

/**
 * The URL checkship gives an application's default service on a target: the first exposed service
 * (alphabetical order) gets `<app>.<target host>` in production and `<app>-staging.<target host>`
 * in staging, with the target's scheme (`domain/config_snapshot.go`, subDomain).
 *
 * Returns undefined when the target has no URL (a manual proxy) or its URL cannot be parsed.
 */
export function appUrl(
	targetUrl: string | undefined,
	name: string,
	environment: Environment = 'production'
): string | undefined {
	if (!targetUrl || !name) return undefined;
	try {
		const target = new URL(targetUrl);
		const suffix = environment === 'staging' ? '-staging' : '';
		return `${target.protocol}//${name}${suffix}.${target.host}`;
	} catch {
		return undefined;
	}
}

/**
 * Whether a browser treats this URL as a secure context: https, or an http address on localhost
 * or a `*.localhost` name. Some apps (Vaultwarden's web vault) only work in one.
 */
export function isSecureContextUrl(url: string | undefined): boolean {
	if (!url) return false;
	try {
		const u = new URL(url);
		if (u.protocol === 'https:') return true;
		const host = u.hostname;
		return host === 'localhost' || host.endsWith('.localhost') || host === '127.0.0.1';
	} catch {
		return false;
	}
}
