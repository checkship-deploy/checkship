import appsService from '#lib/resources/apps.ts';
import targetsService from '#lib/resources/targets.ts';

export const load = async ({ fetch, depends }) => {
	const [targets, apps] = await Promise.all([
		targetsService.fetchAll({ active_only: true }, { fetch, depends }),
		appsService.fetchAll({ fetch, depends })
	]);

	return {
		targets,
		apps
	};
};
