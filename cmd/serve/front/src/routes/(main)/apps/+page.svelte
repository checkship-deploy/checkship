<!--
	Applications (DESIGN.md §5 Lists, §11.1): one list row per application. The tile and the badge
	carry the latest production deployment; the details line has the URL, target, staging and when it
	last changed. The row opens the application; the ⋯ menu has Open, Deploy, Logs, Settings, Copy
	URL and Delete. Filter by name and target.
-->
<script lang="ts">
	import CommandLineIcon from '@hugeicons/core-free-icons/CommandLineIcon';
	import Copy01Icon from '@hugeicons/core-free-icons/Copy01Icon';
	import DashboardSquare01Icon from '@hugeicons/core-free-icons/DashboardSquare01Icon';
	import Delete02Icon from '@hugeicons/core-free-icons/Delete02Icon';
	import Key01Icon from '@hugeicons/core-free-icons/Key01Icon';
	import Loading03Icon from '@hugeicons/core-free-icons/Loading03Icon';
	import Rocket01Icon from '@hugeicons/core-free-icons/Rocket01Icon';
	import Search01Icon from '@hugeicons/core-free-icons/Search01Icon';
	import ServerStack01Icon from '@hugeicons/core-free-icons/ServerStack01Icon';
	import Settings02Icon from '@hugeicons/core-free-icons/Settings02Icon';
	import ViewIcon from '@hugeicons/core-free-icons/ViewIcon';
	import { toast } from 'svelte-sonner';
	import { appUrl as urlOf } from '#lib/app-url.ts';
	import { copyText } from '#lib/clipboard.ts';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import EmptyState from '#lib/components/empty-state.svelte';
	import docs from '#lib/docs.ts';
	import Icon from '#lib/components/icon.svelte';
	import NewAppMenu from '#lib/components/new-app-menu.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import TimelineList from '#lib/components/timeline/timeline-list.svelte';
	import TimelineRow from '#lib/components/timeline/timeline-row.svelte';
	import FilterMenu from '#lib/components/table/filter-menu.svelte';
	import RowMenuGroup from '#lib/components/table/row-menu-group.svelte';
	import RowMenuItem from '#lib/components/table/row-menu-item.svelte';
	import RowMenu from '#lib/components/table/row-menu.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service, { type App } from '#lib/resources/apps.ts';
	import { deploymentTile } from '#lib/deployment-row.ts';
	import type { Deployment } from '#lib/resources/deployments.ts';
	import jobsService from '#lib/resources/jobs.ts';
	import { deploymentBadge, deploymentMoment, isFailure } from '#lib/status.ts';
	import { templateOfApp } from '#lib/templates/recognise.ts';
	import { hasWebPage } from '#lib/templates/values.ts';
	import TemplateLogo from '#lib/templates/template-logo.svelte';

	const { data: apps } = service.queryAll();
	const { data: jobs } = jobsService.queryAll({ page: 1 });

	const failed = $derived(
		($apps ?? []).filter(
			(app) =>
				!!app.latest_deployments.production && isFailure(app.latest_deployments.production.state)
		).length
	);

	let search = $state('');
	let target = $state('all');

	const targetOptions = $derived([
		{ value: 'all', label: l.translate('detail.filter.any'), icon: ServerStack01Icon },
		...[...new Map(($apps ?? []).map((a) => [a.production_target.id, a.production_target.name]))]
			.sort((a, b) => a[1].localeCompare(b[1]))
			.map(([value, label]) => ({ value, label, icon: ServerStack01Icon }))
	]);

	const shown = $derived(
		($apps ?? []).filter(
			(app) =>
				(target === 'all' ||
					app.production_target.id === target ||
					app.staging_target.id === target) &&
				app.name.toLowerCase().includes(search.trim().toLowerCase())
		)
	);

	/** The default service's URL: <app>.<target host> (how checkship exposes production). */
	function appUrl(app: App): string | undefined {
		return urlOf(app.production_target.url, app.name);
	}

	/** A database or cache from a template publishes no web port: its URL would open nothing. */
	function hasNoPage(app: App): boolean {
		const template = templateOfApp(app);
		return !!template && !hasWebPage(template);
	}

	/** When the application last changed: its most recent deployment, else nothing. */
	function updated(app: App): string | undefined {
		const latest = [app.latest_deployments.production, app.latest_deployments.staging]
			.filter((d) => !!d)
			.map((d) => deploymentMoment(d))
			.sort()
			.at(-1);
		return latest ? l.relative(latest) : undefined;
	}

	function deploymentText(d: Deployment): string {
		return `${deploymentBadge(d.state).label} · #${d.deployment_number}`;
	}

	/** The details line, in reading order: where it lives, staging, when it last changed. */
	function detailsOf(app: App, url: string | undefined): string[] {
		const staging = app.latest_deployments.staging;
		return [
			hasNoPage(app) ? l.translate('apps.row.no_page') : (url ?? l.translate('targets.url.manual')),
			app.production_target.name,
			!app.cleanup_requested_at && staging
				? l.translate('apps.row.staging', [deploymentText(staging)])
				: undefined,
			updated(app)
		].filter((part): part is string => !!part);
	}

	/** Everything the row shows, for its accessible name. */
	function label(app: App, url: string | undefined): string {
		const production = app.latest_deployments.production;
		return [
			app.name,
			app.cleanup_requested_at
				? l.translate('apps.deletion_requested')
				: l.translate('apps.row.production', [
						production ? deploymentText(production) : l.translate('apps.no_deployment')
					]),
			...detailsOf(app, url)
		].join(', ');
	}

	async function copyUrl(url: string) {
		if (await copyText(url)) toast.success(l.translate('copied'));
		else toast.error(l.translate('copy_blocked'));
	}

	let deleting = $state<App>();
	let deleteOpen = $state(false);

	async function remove() {
		await service.delete(deleting!.id);
		await service.fetchAll({ cache: 'no-store' });
	}
</script>

<PageHeader title={l.translate('breadcrumb.applications')}>
	{#snippet actions()}
		<NewAppMenu />
	{/snippet}
	{#snippet status()}
		{#if $apps}
			<span>{l.translate('apps.summary', [$apps.length, failed, $jobs?.total ?? 0])}</span>
		{/if}
	{/snippet}
</PageHeader>

{#if $apps && $apps.length > 0}
	<div class="flex flex-col gap-3">
		<div class="flex flex-wrap items-center justify-between gap-2">
			<div class="flex flex-wrap items-center gap-2">
				<div class="relative w-72 max-w-full">
					<Input
						type="search"
						bind:value={search}
						placeholder={l.translate('apps.filter')}
						aria-label={l.translate('apps.filter')}
						class="pr-9"
					/>
					<Icon
						icon={Search01Icon}
						class="pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2 text-muted-foreground"
					/>
				</div>
				<FilterMenu
					label={l.translate('column.target')}
					icon={ServerStack01Icon}
					bind:value={target}
					options={targetOptions}
				/>
			</div>
		</div>

		{#if shown.length > 0}
			<TimelineList label={l.translate('breadcrumb.applications')}>
				{#each shown as app (app.id)}
					{@const url = appUrl(app)}
					{@const production = app.latest_deployments.production}
					{@const template = templateOfApp(app)}
					{@const tile = app.cleanup_requested_at
						? { icon: Loading03Icon, tone: 'muted' as const, spin: true }
						: production
							? deploymentTile(production.state)
							: { icon: DashboardSquare01Icon, tone: 'muted' as const, spin: false }}
					<!-- An app running a template's image shows that project's logo, and its badge keeps
					     the state. While it deploys or is deleted, the spinner says so instead. -->
					{#snippet logo()}
						{#if template}<TemplateLogo {template} />{/if}
					{/snippet}
					<TimelineRow
						icon={tile.icon}
						tone={tile.tone}
						spin={tile.spin}
						tile={template && !tile.spin ? logo : undefined}
						label={label(app, url)}
						href={routes.app(app.id)}
						muted={!!app.cleanup_requested_at}
					>
						{#snippet title()}{app.name}{/snippet}
						{#snippet badge()}
							{#if app.cleanup_requested_at}
								<StatusBadge state="deleting" label={l.translate('apps.deletion_requested')} />
							{:else if production}
								{@const status = deploymentBadge(production.state)}
								<StatusBadge state={status.state} label={deploymentText(production)} />
							{:else}
								<span class="text-xs font-normal text-muted-foreground"
									>{l.translate('apps.no_deployment')}</span
								>
							{/if}
						{/snippet}
						{#snippet details()}{detailsOf(app, url).join(' · ')}{/snippet}
						{#snippet actions()}
							<RowMenu label={l.translate('table.menu', [app.name])}>
								<RowMenuGroup>
									<RowMenuItem icon={ViewIcon} href={routes.app(app.id)}>
										{l.translate('menu.open')}
									</RowMenuItem>
									{#if !app.cleanup_requested_at}
										{#if template}
											<!-- Where its passwords, tokens and connection string are. -->
											<RowMenuItem icon={Key01Icon} href="{routes.app(app.id)}?access">
												{l.translate('access.open')}
											</RowMenuItem>
										{/if}
										<RowMenuItem icon={Rocket01Icon} href={routes.createDeployment(app.id)}>
											{l.translate('menu.deploy')}
										</RowMenuItem>
										<RowMenuItem icon={CommandLineIcon} href={routes.appLogs(app.id)}>
											{l.translate('menu.logs')}
										</RowMenuItem>
										<RowMenuItem icon={Settings02Icon} href={routes.editApp(app.id)}>
											{l.translate('menu.settings')}
										</RowMenuItem>
									{/if}
									{#if url}
										<RowMenuItem icon={Copy01Icon} onSelect={() => copyUrl(url)}>
											{l.translate('menu.copy_url')}
										</RowMenuItem>
									{/if}
								</RowMenuGroup>
								{#if !app.cleanup_requested_at}
									<RowMenuGroup>
										<RowMenuItem
											icon={Delete02Icon}
											variant="destructive"
											onSelect={() => {
												deleting = app;
												deleteOpen = true;
											}}
										>
											{l.translate('menu.delete')}
										</RowMenuItem>
									</RowMenuGroup>
								{/if}
							</RowMenu>
						{/snippet}
					</TimelineRow>
				{/each}
			</TimelineList>
		{:else}
			<EmptyState
				illustration="search"
				size="section"
				title={l.translate('apps.no_match.title')}
				description={l.translate('apps.no_match.description')}
			>
				{#snippet action()}
					<Button
						variant="outline"
						onclick={() => {
							search = '';
							target = 'all';
						}}
					>
						{l.translate('list.clear_filters')}
					</Button>
				{/snippet}
			</EmptyState>
		{/if}
	</div>
{:else if $apps}
	<EmptyState
		illustration="apps"
		link={{ href: docs.applications, label: l.translate('empty.docs.applications') }}
		title={l.translate('apps.empty.title')}
		description={l.translate('apps.empty.description')}
	>
		{#snippet action()}
			<NewAppMenu align="center" />
		{/snippet}
	</EmptyState>
{/if}

{#if deleting}
	<ConfirmDialog
		bind:open={deleteOpen}
		destructive
		icon={Delete02Icon}
		title={l.translate('app.delete.title', [deleting.name])}
		description={l.translate('app.delete.description')}
		confirmLabel={l.translate('app.delete')}
		typeToConfirm={deleting.name}
		typeLabel={l.translate('app.delete.type_name', [deleting.name])}
		onconfirm={remove}
	/>
{/if}
