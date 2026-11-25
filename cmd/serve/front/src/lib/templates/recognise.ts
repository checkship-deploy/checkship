import type { LatestDeployments } from '#lib/resources/apps.ts';
import type { SourceData } from '#lib/resources/deployments.ts';
import { templates } from './catalog.ts';
import type { Template } from './types.ts';

/**
 * Which template an application runs, so lists can show the project's logo. checkship stores no
 * link from an app to a template: an app is recognised by its latest deployment's compose file,
 * when that file runs a single image, built by nobody, whose repository is a template's. An app
 * deployed from scratch with `postgres:17` alone gets PostgreSQL's logo too, which is what it
 * runs; one with an API and a database beside it gets none, since the logo would name only half.
 */

/** An image reference's repository: no registry default, tag or digest, `library/` dropped. */
export function imageRepository(image: string): string {
	let ref = image.trim().replace(/^["']|["']$/g, '');
	ref = ref.split('@')[0];
	const colon = ref.lastIndexOf(':');
	if (colon > ref.lastIndexOf('/')) ref = ref.slice(0, colon);
	return ref
		.replace(/^(index\.)?docker\.io\//, '')
		.replace(/^library\//, '')
		.toLowerCase();
}

/** The template a deployment source runs, if it is one of theirs. */
export function templateOfSource(source: SourceData | undefined): Template | undefined {
	if (source?.discriminator !== 'raw') return undefined;
	if (/^\s*build\s*:/m.test(source.data)) return undefined;
	const images = [...source.data.matchAll(/^\s*image\s*:\s*(\S+)/gm)].map((m) =>
		imageRepository(m[1])
	);
	if (images.length !== 1) return undefined;
	return templates.find((t) => imageRepository(t.image) === images[0]);
}

/** The template an application runs: from production's latest deployment, else staging's. */
export function templateOfApp(app: {
	latest_deployments: LatestDeployments<{ source: SourceData }>;
}): Template | undefined {
	const { production, staging } = app.latest_deployments;
	return templateOfSource((production ?? staging)?.source);
}
