<!--
	Application header shared by every tab (DESIGN.md §11.7): the name, a status line (what each
	environment serves, the repository), Visit and a split Deploy button, then the tabs. The header
	never changes between tabs; Settings is a tab, not a second button. It owns the New deployment and deployment preview drawers and hands the pages a few
	actions to open them.

	Deploy is one click (Coolify, Render): the latest commit of production's branch. Its arrow opens
	the other ways: staging, a compose file, an archive, redeploying production.
-->
<script lang="ts" module>
	import type { Environment, SourceDataDiscriminator } from '#lib/resources/deployments.ts';

	export type AppActions = {
		/** Open the New deployment drawer, preset. */
		openDeploy: (environment?: Environment, kind?: SourceDataDiscriminator) => void;
		/** Deploy the latest commit of an environment's branch, or open the drawer when unknown. */
		quickDeploy: (environment: Environment) => void;
		openPreview: (number: number) => void;
		previewNumber: () => number | undefined;
	};
</script>

<script lang="ts">
	import ArrowDown01Icon from '@hugeicons/core-free-icons/ArrowDown01Icon';
	import File01Icon from '@hugeicons/core-free-icons/File01Icon';
	import GitBranchIcon from '@hugeicons/core-free-icons/GitBranchIcon';
	import Key01Icon from '@hugeicons/core-free-icons/Key01Icon';
	import LinkSquare02Icon from '@hugeicons/core-free-icons/LinkSquare02Icon';
	import { page } from '$app/state';
	import Loading03Icon from '@hugeicons/core-free-icons/Loading03Icon';
	import RepeatIcon from '@hugeicons/core-free-icons/RepeatIcon';
	import Rocket01Icon from '@hugeicons/core-free-icons/Rocket01Icon';
	import Upload04Icon from '@hugeicons/core-free-icons/Upload04Icon';
	import type { Snippet } from 'svelte';
	import { toast } from 'svelte-sonner';
	import Drawer from '#lib/components/drawer.svelte';
	import Icon from '#lib/components/icon.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import type { AppDetail } from '#lib/resources/apps.ts';
	import service, { DeploymentStatus, type Deployment } from '#lib/resources/deployments.ts';
	import { isCancelled, isFailure } from '#lib/status.ts';
	import { templateOfApp } from '#lib/templates/recognise.ts';
	import TemplateLogo from '#lib/templates/template-logo.svelte';
	import { cn } from '#lib/utils.js';
	import DeploymentPreview from './deployment-preview.svelte';
	import DeployForm from './deployments/deploy-form.svelte';
	import AppTabs from './app-tabs.svelte';
	import AccessDrawer from './access-drawer.svelte';
	import { branchOf, liveUrl, socketAddresses } from './app-state.ts';

	type Props = {
		app: AppDetail;
		apiKey: string;
		tab: 'overview' | 'deployments' | 'logs' | 'backups' | 'settings';
		children: Snippet<[AppActions]>;
	};

	let { app, apiKey, tab, children }: Props = $props();

	const deleting = $derived(!!app.cleanup_requested_at);
	const url = $derived(liveUrl(app.latest_deployments.production));
	// With a custom domain (seelf#5), the button names it: that's where it goes.
	const visitDomain = $derived(
		app.latest_deployments.production?.state.services?.some((s) => s.domains?.length) && url
			? new URL(url).host
			: undefined
	);

	let deployOpen = $state(false);
	let deployEnvironment = $state<Environment>('production');
	let deployKind = $state<SourceDataDiscriminator | undefined>();
	let deployLoading = $state(false);
	let quickLoading = $state(false);
	let previewOpen = $state(false);
	let previewNumber = $state(0);

	function openDeploy(environment: Environment = 'production', kind?: SourceDataDiscriminator) {
		deployEnvironment = environment;
		deployKind = kind;
		deployOpen = true;
	}

	function openPreview(number: number) {
		previewNumber = number;
		previewOpen = true;
	}

	async function quickDeploy(environment: Environment) {
		const branch = branchOf(app, environment);
		if (!app.version_control || !branch) {
			openDeploy(environment, app.version_control ? 'git' : undefined);
			return;
		}
		quickLoading = true;
		try {
			const deployment = await service.queue(app.id, { environment, git: { branch } });
			openPreview(deployment.deployment_number);
		} catch (ex) {
			toast.error(ex instanceof Error ? ex.message : l.translate('unexpected_error'));
		} finally {
			quickLoading = false;
		}
	}

	async function redeployProduction() {
		const latest = app.latest_deployments.production;
		if (!latest) return;
		try {
			const deployment = await service.redeploy(app.id, latest.deployment_number);
			openPreview(deployment.deployment_number);
		} catch (ex) {
			toast.error(ex instanceof Error ? ex.message : l.translate('unexpected_error'));
		}
	}

	function queued(deployment: Deployment) {
		deployOpen = false;
		openPreview(deployment.deployment_number);
	}

	const actions: AppActions = {
		openDeploy,
		quickDeploy,
		openPreview,
		previewNumber: () => (previewOpen ? previewNumber : undefined)
	};

	type EnvState = 'live' | 'deploying' | 'failed' | 'cancelled' | 'none';
	function envState(environment: Environment): EnvState {
		const latest = app.latest_deployments[environment];
		if (!latest) return 'none';
		if (latest.state.status === DeploymentStatus.Succeeded) return 'live';
		if (isFailure(latest.state)) return 'failed';
		if (isCancelled(latest.state)) return 'cancelled';
		return 'deploying';
	}

	const productionBranch = $derived(branchOf(app, 'production'));
	const stagingBranch = $derived(branchOf(app, 'staging'));
	/** The template the app runs, recognised from its compose file, for the logo by its name. */
	const template = $derived(templateOfApp(app));
	/**
	 * Sign-in and connection: for an app from a template, or any app reached over tcp or udp.
	 * ?access on the URL opens it, as the Applications row menu does.
	 */
	const sockets = $derived(
		socketAddresses(app.latest_deployments.production, app.name, 'production')
	);
	const hasAccess = $derived(
		!!template ||
			sockets.length > 0 ||
			socketAddresses(app.latest_deployments.staging, app.name, 'staging').length > 0
	);
	let accessOpen = $state(page.url.searchParams.has('access'));
</script>

<PageHeader
	title={app.name}
	crumbs={[{ href: routes.apps, label: l.translate('breadcrumb.applications') }]}
>
	{#snippet leading()}
		{#if template}<TemplateLogo {template} class="size-7" />{/if}
	{/snippet}
	{#snippet actions()}
		{#if !deleting}
			<Button
				variant="outline"
				href={url}
				target="_blank"
				rel="noopener noreferrer"
				disabled={!url}
				title={url ??
					(sockets.length > 0 ? l.translate('detail.visit.tcp') : l.translate('detail.visit.none'))}
			>
				<Icon icon={LinkSquare02Icon} />
				{visitDomain
					? l.translate('detail.visit.domain', [visitDomain])
					: l.translate('detail.visit')}
			</Button>
			{#if hasAccess}
				<Button variant="outline" onclick={() => (accessOpen = true)}>
					<Icon icon={Key01Icon} />
					{l.translate('access.open')}
				</Button>
			{/if}
			<div class="flex gap-0.5">
				<Button
					class="rounded-r-none"
					loading={quickLoading}
					title={productionBranch
						? l.translate('detail.deploy.hint', [productionBranch])
						: undefined}
					onclick={() => quickDeploy('production')}
				>
					<Icon icon={Rocket01Icon} />
					{l.translate('detail.deploy')}
				</Button>
				<DropdownMenu.Root>
					<DropdownMenu.Trigger>
						{#snippet child({ props })}
							<Button
								{...props}
								size="icon"
								class="w-8 rounded-l-none"
								aria-label={l.translate('detail.deploy.more')}
							>
								<Icon icon={ArrowDown01Icon} />
							</Button>
						{/snippet}
					</DropdownMenu.Trigger>
					<DropdownMenu.Content align="end" sideOffset={6} class="w-74 p-1.5">
						{#if app.version_control}
							<DropdownMenu.Label
								class="px-2.5 pt-2 pb-1 text-xs font-semibold text-muted-foreground"
							>
								{l.translate('detail.menu.git')}
							</DropdownMenu.Label>
							{#each ['production', 'staging'] as const as environment (environment)}
								{@const branch = environment === 'production' ? productionBranch : stagingBranch}
								<DropdownMenu.Item
									class="min-h-11 gap-2.5 px-2.5 py-1.5"
									onSelect={() => quickDeploy(environment)}
								>
									<Icon icon={Rocket01Icon} class="text-muted-foreground" />
									<span class="flex flex-col">
										<span class="text-label font-medium"
											>{l.translate(`detail.menu.deploy.${environment}`)}</span
										>
										<span class="text-xs text-muted-foreground">
											{branch
												? l.translate('detail.menu.latest', [branch])
												: l.translate('detail.menu.pick_branch')}
										</span>
									</span>
								</DropdownMenu.Item>
							{/each}
						{/if}
						<DropdownMenu.Label
							class="px-2.5 pt-2 pb-1 text-xs font-semibold text-muted-foreground"
						>
							{l.translate('detail.menu.other')}
						</DropdownMenu.Label>
						<DropdownMenu.Item
							class="min-h-11 gap-2.5 px-2.5 py-1.5"
							onSelect={() => openDeploy('production', 'raw')}
						>
							<Icon icon={File01Icon} class="text-muted-foreground" />
							<span class="flex flex-col">
								<span class="text-label font-medium">{l.translate('detail.menu.compose')}</span>
								<span class="text-xs text-muted-foreground"
									>{l.translate('detail.menu.compose.hint')}</span
								>
							</span>
						</DropdownMenu.Item>
						<DropdownMenu.Item
							class="min-h-11 gap-2.5 px-2.5 py-1.5"
							onSelect={() => openDeploy('production', 'archive')}
						>
							<Icon icon={Upload04Icon} class="text-muted-foreground" />
							<span class="flex flex-col">
								<span class="text-label font-medium">{l.translate('detail.menu.archive')}</span>
								<span class="text-xs text-muted-foreground"
									>{l.translate('detail.menu.archive.hint')}</span
								>
							</span>
						</DropdownMenu.Item>
						{#if app.latest_deployments.production}
							<DropdownMenu.Label
								class="px-2.5 pt-2 pb-1 text-xs font-semibold text-muted-foreground"
							>
								{l.translate('detail.menu.again')}
							</DropdownMenu.Label>
							<DropdownMenu.Item
								class="min-h-11 gap-2.5 px-2.5 py-1.5"
								onSelect={redeployProduction}
							>
								<Icon icon={RepeatIcon} class="text-muted-foreground" />
								<span class="flex flex-col">
									<span class="text-label font-medium">
										{l.translate('detail.menu.redeploy', [
											app.latest_deployments.production.deployment_number
										])}
									</span>
									<span class="text-xs text-muted-foreground"
										>{l.translate('detail.menu.redeploy.hint')}</span
									>
								</span>
							</DropdownMenu.Item>
						{/if}
					</DropdownMenu.Content>
				</DropdownMenu.Root>
			</div>
		{/if}
	{/snippet}
	{#snippet status()}
		{#each ['production', 'staging'] as const as environment (environment)}
			{@const state = envState(environment)}
			{@const latest = app.latest_deployments[environment]}
			<span class="flex items-center gap-2 text-ink">
				{#if state === 'deploying'}
					<Icon icon={Loading03Icon} class="size-3.5 animate-spin text-warning" />
				{:else}
					<span
						class={cn(
							'size-2 rounded-full',
							state === 'live' && 'bg-success',
							state === 'failed' && 'bg-danger',
							(state === 'none' || state === 'cancelled') && 'bg-faint'
						)}
					></span>
				{/if}
				{l.translate(`detail.state.${environment}.${state}`)}{#if latest}&nbsp;·&nbsp;<span
						class="font-mono text-xs">#{latest.deployment_number}</span
					>{/if}
			</span>
		{/each}
		{#if app.version_control}
			<span class="flex min-w-0 items-center gap-1.5">
				<Icon icon={GitBranchIcon} class="size-3.5" />
				<span class="truncate font-mono text-xs"
					>{app.version_control.url.replace(/^https?:\/\//, '')}</span
				>{#if productionBranch}<span class="shrink-0">· {productionBranch}</span>{/if}
			</span>
		{/if}
		<span>{l.translate('app.created', [l.date(app.created_at), app.created_by.email])}</span>
	{/snippet}
</PageHeader>

<div class="-mt-3 mb-7">
	<AppTabs appId={app.id} active={tab} />
</div>

{@render children(actions)}

<Drawer
	bind:open={deployOpen}
	title={l.translate('deployment.new')}
	description={l.translate('deploy.description', [app.name])}
>
	{#if deployOpen}
		<DeployForm
			id="deploy-drawer"
			{app}
			{apiKey}
			environment={deployEnvironment}
			kind={deployKind}
			bind:loading={deployLoading}
			onqueued={queued}
		/>
	{/if}
	{#snippet footer()}
		<Button variant="outline" onclick={() => (deployOpen = false)}>{l.translate('cancel')}</Button>
		<Button type="submit" form="deploy-drawer" loading={deployLoading}>
			<Icon icon={Rocket01Icon} />
			{l.translate('deployment.deploy')}
		</Button>
	{/snippet}
</Drawer>

<AccessDrawer bind:open={accessOpen} {app} {template} />

{#if previewNumber > 0}
	<DeploymentPreview {app} bind:number={previewNumber} bind:open={previewOpen} />
{/if}
