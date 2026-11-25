/**
 * Light checks on a compose file before it is deployed (the Compose file tab and the template
 * drawer). The dashboard ships no YAML parser: these catch what stops a file before checkship's own
 * reading (compose-go, in the deployment), which reports anything deeper in the deployment log.
 */

export type ComposeProblem =
	'compose.problem.empty' | 'compose.problem.tabs' | 'compose.problem.no_services';

/** What is wrong with a compose file, in the order a person would fix it. */
export function composeProblem(compose: string): ComposeProblem | undefined {
	if (!compose.trim()) return 'compose.problem.empty';
	// YAML forbids tabs for indentation: a line starting with one is never valid there.
	if (/^\t/m.test(compose)) return 'compose.problem.tabs';
	if (!/^services:\s*(#.*)?$/m.test(compose)) return 'compose.problem.no_services';
	return undefined;
}

/** The service names under `services:`, as written (two-space or deeper indented keys). */
export function composeServices(compose: string): string[] {
	const lines = compose.split('\n');
	const start = lines.findIndex((l) => /^services:\s*(#.*)?$/.test(l));
	if (start < 0) return [];
	const names: string[] = [];
	let indent: number | undefined;
	for (const line of lines.slice(start + 1)) {
		if (!line.trim() || /^\s*#/.test(line)) continue;
		if (/^\S/.test(line)) break;
		const match = /^(\s+)([\w.-]+):\s*(#.*)?$/.exec(line);
		if (!match) continue;
		indent ??= match[1].length;
		if (match[1].length === indent) names.push(match[2]);
	}
	return names;
}
