<!--
	Compose file tab (DESIGN.md §11.7): the compose file an environment runs, to read, edit and
	deploy again. checkship keeps no compose file apart from deployments: the file shown is the
	latest deployment's own (a raw source), and Save and deploy queues a new deployment of the
	edited file to that environment, then opens it. An environment deployed from Git or an archive
	has its compose file there: the tab says so and lets you write one to deploy instead. Each
	environment keeps its draft while you switch between them.
-->
<script lang="ts">
	import ArrowTurnBackwardIcon from '@hugeicons/core-free-icons/ArrowTurnBackwardIcon';
	import Copy01Icon from '@hugeicons/core-free-icons/Copy01Icon';
	import Download04Icon from '@hugeicons/core-free-icons/Download04Icon';
	import Rocket01Icon from '@hugeicons/core-free-icons/Rocket01Icon';
	import { toast } from 'svelte-sonner';
	import { goto } from '$app/navigation';
	import { copyText } from '#lib/clipboard.ts';
	import CodeEditor from '#lib/components/code-editor.svelte';
	import FieldMessage from '#lib/components/field-message.svelte';
	import Icon from '#lib/components/icon.svelte';
	import Notice from '#lib/components/notice.svelte';
	import Segmented from '#lib/components/segmented.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { composeProblem, composeServices } from '#lib/compose.ts';
	import { GLOBAL_ERROR_NAME, submitter } from '#lib/form.ts';
	import l, { type AppTranslationsString } from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import appsService from '#lib/resources/apps.ts';
	import deploymentsService, { type Environment } from '#lib/resources/deployments.ts';
	import { templateOfApp } from '#lib/templates/recognise.ts';
	import AppWorkspace from '../app-workspace.svelte';

	let { data } = $props();

	const appQuery = $derived(appsService.queryById(data.app.id));
	const appStore = $derived(appQuery.data);
	const app = $derived($appStore ?? data.app);

	let environment = $state<Environment>('production');
	const environments = [
		{ value: 'production' as const, label: l.translate('deploy.environment.production') },
		{ value: 'staging' as const, label: l.translate('deploy.environment.staging') }
	];

	const other = (env: Environment): Environment =>
		env === 'production' ? 'staging' : 'production';

	/** What an environment runs: its latest deployment's compose file, when it deployed one. */
	function deployed(env: Environment) {
		const latest = app.latest_deployments[env];
		if (!latest) return { kind: 'none' as const };
		const source = latest.source;
		if (source.discriminator === 'raw') {
			return { kind: 'raw' as const, compose: source.data, number: latest.deployment_number };
		}
		if (source.discriminator === 'git') {
			return { kind: 'git' as const, branch: source.data.branch, number: latest.deployment_number };
		}
		return { kind: 'archive' as const, number: latest.deployment_number };
	}

	/** Where the editor starts: this environment's file, else the other one's, else nothing. */
	function starting(env: Environment): { compose: string; from?: Environment } {
		const own = deployed(env);
		if (own.kind === 'raw') return { compose: own.compose };
		if (own.kind === 'none') {
			const theirs = deployed(other(env));
			if (theirs.kind === 'raw') return { compose: theirs.compose, from: other(env) };
		}
		return { compose: '' };
	}

	// One draft per environment, kept while you switch between them.
	let drafts = $state<Partial<Record<Environment, string>>>({});
	const current = $derived(deployed(environment));
	const start = $derived(starting(environment));
	const compose = $derived(drafts[environment] ?? start.compose);
	const changed = $derived(
		drafts[environment] !== undefined && drafts[environment] !== start.compose
	);
	const problem = $derived(composeProblem(compose));
	const services = $derived(composeServices(compose));
	const template = $derived(templateOfApp(app));
	let tried = $state(false);

	const { submit, loading, errors } = submitter(async () => {
		const deployment = await deploymentsService.queue(app.id, { environment, raw: compose });
		drafts[environment] = undefined;
		await goto(routes.deployment(app.id, deployment.deployment_number));
	});
	const globalError = $derived($errors?.[GLOBAL_ERROR_NAME] ?? $errors?.raw);

	function onsubmit(event: SubmitEvent) {
		event.preventDefault();
		tried = true;
		if (problem || $loading) return;
		submit().catch(() => {});
	}

	async function copy() {
		if (await copyText(compose)) toast.success(l.translate('copied'));
		else toast.error(l.translate('copy_blocked'));
	}

	function download() {
		const url = URL.createObjectURL(new Blob([compose], { type: 'application/yaml' }));
		const link = document.createElement('a');
		link.href = url;
		link.download = `${app.name}-${environment}.compose.yml`;
		link.click();
		URL.revokeObjectURL(url);
	}
</script>

<AppWorkspace {app} apiKey={data.user.api_key} tab="compose">
	<form class="flex flex-col gap-4" novalidate {onsubmit}>
		<div class="flex flex-wrap items-end justify-between gap-3">
			<div class="flex min-w-0 flex-col gap-1">
				<h2 id="compose-title" class="text-xs font-semibold tracking-wide text-muted-foreground">
					{l.translate('compose.title')}
				</h2>
				<p class="text-label text-muted-foreground">{l.translate('compose.description')}</p>
			</div>
			<span id="compose-env" class="sr-only">{l.translate('dashboard.environment.label')}</span>
			<Segmented bind:value={environment} items={environments} labelledby="compose-env" />
		</div>

		{#if current.kind === 'raw'}
			<p class="text-sm text-muted-foreground">
				{l.translate('compose.from_deployment', [
					current.number,
					l.translate(`deploy.environment.${environment}`)
				])}
			</p>
		{:else if current.kind === 'git'}
			<Notice>
				{l.translate('compose.from_git', [
					l.translate(`deploy.environment.${environment}`),
					current.branch
				])}
			</Notice>
		{:else if current.kind === 'archive'}
			<Notice>
				{l.translate('compose.from_archive', [l.translate(`deploy.environment.${environment}`)])}
			</Notice>
		{:else if start.from}
			<p class="text-sm text-muted-foreground">
				{l.translate('compose.from_other', [
					l.translate(`deploy.environment.${environment}`),
					l.translate(`deploy.environment.${start.from}`)
				])}
			</p>
		{:else}
			<p class="text-sm text-muted-foreground">
				{l.translate('compose.none', [l.translate(`deploy.environment.${environment}`)])}
			</p>
		{/if}

		{#if template}
			<Notice>{l.translate('compose.template', [template.name])}</Notice>
		{/if}

		{#if globalError}
			<Notice tone="danger">{l.translate(globalError as AppTranslationsString)}</Notice>
		{/if}

		<div class="flex flex-col gap-2">
			<div class="flex flex-wrap items-center justify-between gap-2">
				<span class="text-sm font-medium" id="compose-editor-label">compose.yml</span>
				<div class="flex flex-wrap items-center gap-1.5">
					{#if changed}
						<span class="text-xs font-medium text-warning">{l.translate('compose.changed')}</span>
						<Button
							variant="ghost"
							size="sm"
							onclick={() => (drafts[environment] = undefined)}
							disabled={$loading}
						>
							<Icon icon={ArrowTurnBackwardIcon} />
							{l.translate('compose.reset')}
						</Button>
					{/if}
					<Button variant="ghost" size="sm" onclick={copy} disabled={!compose}>
						<Icon icon={Copy01Icon} />
						{l.translate('compose.copy')}
					</Button>
					<Button variant="ghost" size="sm" onclick={download} disabled={!compose}>
						<Icon icon={Download04Icon} />
						{l.translate('compose.download')}
					</Button>
				</div>
			</div>
			<CodeEditor
				value={compose}
				oninput={(event) =>
					(drafts[environment] = (event.currentTarget as HTMLTextAreaElement).value)}
				lang="yaml"
				rows={18}
				spellcheck={false}
				aria-labelledby="compose-editor-label"
				aria-invalid={(tried && !!problem) || undefined}
				aria-describedby="compose-message"
				disabled={$loading}
			/>
			<FieldMessage id="compose-message" error={tried ? problem : undefined}>
				{#snippet help()}
					{#if services.length > 0}
						{l.translate('compose.services', [services.join(', ')])}
					{:else}
						{l.translate('compose.help')}
					{/if}
				{/snippet}
			</FieldMessage>
		</div>

		<div class="flex flex-wrap items-center justify-end gap-2">
			<span class="mr-auto text-xs text-muted-foreground">{l.translate('compose.deploy_hint')}</span
			>
			<Button type="submit" loading={$loading} disabled={!compose.trim()}>
				<Icon icon={Rocket01Icon} />
				{l.translate('compose.deploy', [l.translate(`deploy.environment.${environment}`)])}
			</Button>
		</div>
	</form>
</AppWorkspace>
