import type { AppDetail } from '#lib/resources/apps.ts';
import type { DeploymentDetail, Environment } from '#lib/resources/deployments.ts';

/** The branch an environment last deployed from Git, if any. */
export function branchOf(app: AppDetail, environment: Environment): string | undefined {
	const source = app.latest_deployments[environment]?.source;
	return source?.discriminator === 'git' ? source.data.branch : undefined;
}

/**
 * The URL of the first exposed service of a deployment: its primary custom domain when it has
 * one (seelf#5), else its default address.
 */
export function liveUrl(deployment?: DeploymentDetail): string | undefined {
	return liveUrls(deployment)[0];
}

/** Every http address of the main service: custom domains first, then the default one. */
export function liveUrls(deployment?: DeploymentDetail): string[] {
	for (const service of deployment?.state.services ?? []) {
		if (service.urls?.length) return service.urls;
		// Only a web address can be visited: a database's tcp:// one opens nothing in a browser.
		const url = service.entrypoints.find((e) => e.url && e.router === 'http')?.url;
		if (url) return [url];
	}
	return [];
}

/** A service's tcp or udp entrypoint: where it is reached from outside and from other apps. */
export type SocketAddress = {
	service: string;
	/** `tcp://<target host>:<published port>`, once the target assigned the port. */
	outside?: string;
	/** `<service>.<app>[-staging]:<container port>` on the target's network. */
	inside: string;
};

/**
 * The tcp and udp entrypoints of a deployment: a database or a broker has no web address, but
 * these. Other apps on the target reach a service at its alias, `<service>.<app>` (-staging in
 * staging), on the container's own port.
 */
export function socketAddresses(
	deployment: DeploymentDetail | undefined,
	appName: string,
	environment: Environment
): SocketAddress[] {
	const suffix = environment === 'staging' ? '-staging' : '';
	return (deployment?.state.services ?? []).flatMap((service) =>
		service.entrypoints
			.filter((e) => e.router === 'tcp' || e.router === 'udp')
			.map((e) => ({
				service: service.name,
				outside: e.url,
				inside: `${service.name}.${appName}${suffix}:${e.port}`
			}))
	);
}
