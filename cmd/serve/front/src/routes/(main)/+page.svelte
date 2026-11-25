<!--
	Overview (DESIGN.md §11.10): a bento grid of sections on the page, no tiles or boxes (§1 Cardless), 40px apart. A stats strip compares the period with the previous
	one; the deployments chart is the hero; then what needs attention, infrastructure, the live
	activity feed, applications, durations, environments and failures.

	Live: one EventSource (GET /api/v1/events) sends hints; each hint (debounced) refetches what the
	page shows. The page also refreshes every 30 s, since some server changes send no hint.
-->
<script lang="ts">
	import Layers01Icon from '@hugeicons/core-free-icons/Layers01Icon';
	import TestTube01Icon from '@hugeicons/core-free-icons/TestTube01Icon';
	import Rocket01Icon from '@hugeicons/core-free-icons/Rocket01Icon';
	import Calendar03Icon from '@hugeicons/core-free-icons/Calendar03Icon';
	import ArrowRight01Icon from '@hugeicons/core-free-icons/ArrowRight01Icon';
	import CancelCircleIcon from '@hugeicons/core-free-icons/CancelCircleIcon';
	import Tick02Icon from '@hugeicons/core-free-icons/Tick02Icon';
	import WifiDisconnected01Icon from '@hugeicons/core-free-icons/WifiDisconnected01Icon';
	import { onMount, untrack } from 'svelte';
	import CalendarHeatmap from '#lib/components/charts/calendar-heatmap.svelte';
	import ShareBar, { type ShareItem } from '#lib/components/charts/share-bar.svelte';
	import ActivityFeed from '#lib/components/dashboard/activity-feed.svelte';
	import DeployCharts from '#lib/components/dashboard/deploy-charts.svelte';
	import AppsTile from '#lib/components/dashboard/apps-tile.svelte';
	import AttentionTile from '#lib/components/dashboard/attention-tile.svelte';
	import FailuresDrawer from '#lib/components/dashboard/failures-drawer.svelte';
	import InfraTile from '#lib/components/dashboard/infra-tile.svelte';
	import StatsStrip from '#lib/components/dashboard/stats-strip.svelte';
	import Tile from '#lib/components/dashboard/tile.svelte';
	import Icon from '#lib/components/icon.svelte';
	import Illustration from '#lib/components/illustration.svelte';
	import NewAppMenu from '#lib/components/new-app-menu.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import SectionHeading from '#lib/components/section-heading.svelte';
	import Segmented from '#lib/components/segmented.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import { connectLive, liveChanges, liveStatus } from '#lib/live.ts';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import appsService from '#lib/resources/apps.ts';
	import dashboard, {
		DASHBOARD_FALLBACK_POLL_MS,
		type DashboardFilters,
		type DashboardRange
	} from '#lib/resources/dashboard.ts';
	import type { Environment } from '#lib/resources/deployments.ts';
	import jobsService from '#lib/resources/jobs.ts';
	import targetsService from '#lib/resources/targets.ts';
	import EmptyState from '#lib/components/empty-state.svelte';

	let range = $state<DashboardRange>(30);
	let environment = $state<'' | Environment>('');
	const filters = $derived<DashboardFilters>({ range, environment: environment || undefined });

	const summaryQuery = $derived(dashboard.querySummary(filters));
	const summaryData = $derived(summaryQuery.data);
	const summaryError = $derived(summaryQuery.error);
	const summary = $derived($summaryData);

	const { data: apps } = appsService.queryAll();
	const { data: targets } = targetsService.queryAll();
	const { data: runningJobs } = jobsService.queryAll({ page: 1, status: 'running' }, false);
	const { data: pendingJobs } = jobsService.queryAll({ page: 1, status: 'pending' }, false);
	const { data: failedJobs } = jobsService.queryAll({ page: 1, status: 'failed' }, false);

	/** Incremented on every live hint and every fallback tick; tiles refetch on change. */
	let refresh = $state(0);
	let failuresOpen = $state(false);

	const appName = (id: string) => $apps?.find((a) => a.id === id)?.name;

	async function refetch() {
		await Promise.allSettled([
			dashboard.fetchSummary(filters, { cache: 'no-store' }),
			appsService.fetchAll({ cache: 'no-store' }),
			targetsService.fetchAll(undefined, { cache: 'no-store' }),
			jobsService.fetchAll({ page: 1, status: 'running' }, { cache: 'no-store' }),
			jobsService.fetchAll({ page: 1, status: 'pending' }, { cache: 'no-store' }),
			jobsService.fetchAll({ page: 1, status: 'failed' }, { cache: 'no-store' })
		]);
	}

	onMount(() => {
		const disconnect = connectLive();
		const timer = setInterval(() => {
			refresh++;
			refetch();
		}, DASHBOARD_FALLBACK_POLL_MS);
		return () => {
			disconnect();
			clearInterval(timer);
		};
	});

	$effect(() => {
		if ($liveChanges.seq === 0) return;
		untrack(() => {
			refresh++;
			refetch();
		});
	});

	type RangeKey = '7' | '30' | '90';
	const ranges: { value: RangeKey; label: string }[] = [
		{ value: '7', label: l.translate('charts.days', [7]) },
		{ value: '30', label: l.translate('charts.days', [30]) },
		{ value: '90', label: l.translate('charts.days', [90]) }
	];
	const environments: { value: '' | Environment; label: string }[] = [
		{ value: '', label: l.translate('dashboard.environment.all') },
		{ value: 'production', label: l.translate('deploy.environment.production') },
		{ value: 'staging', label: l.translate('deploy.environment.staging') }
	];

	const firstRun = $derived($apps !== undefined && $apps.length === 0);
	const activeTargets = $derived(($targets ?? []).filter((t) => !t.cleanup_requested_at).length);

	// The calendar always looks back 13 weeks: the 90-day summary (the same query when the range is 90).
	const calendarFilters = $derived<DashboardFilters>({
		range: 90,
		environment: filters.environment
	});
	const calendarQuery = $derived(dashboard.querySummary(calendarFilters));
	const calendarData = $derived(calendarQuery.data);
	const calendarError = $derived(calendarQuery.error);
	const calendar = $derived($calendarData);

	const percent = (part: number, whole: number) =>
		`${whole ? Math.round((part / whole) * 100) : 0}%`;

	const environmentShare = $derived.by((): ShareItem[] => {
		const stats = (summary?.by_environment ?? []).filter((e) => e.deployments > 0);
		const total = stats.reduce((sum, e) => sum + e.deployments, 0);
		return stats.map((e) => ({
			key: e.environment,
			label: l.translate(`deploy.environment.${e.environment}`),
			value: e.deployments,
			color:
				e.environment === 'production'
					? 'var(--accent)'
					: 'color-mix(in oklab, var(--accent) 50%, var(--ink))',
			cells: [
				String(e.deployments),
				percent(e.deployments, total),
				l.translate('dashboard.env.success', [percent(e.succeeded, e.succeeded + e.failed)])
			]
		}));
	});

	const errorShade = [
		'var(--danger)',
		'color-mix(in oklab, var(--danger) 65%, var(--surface))',
		'color-mix(in oklab, var(--danger) 40%, var(--surface))',
		'var(--faint)'
	];
	const errorShare = $derived.by((): ShareItem[] => {
		const errors = summary?.errors ?? [];
		const total = errors.reduce((sum, e) => sum + e.count, 0);
		return errors.map((e, i) => ({
			key: e.code,
			label: e.code,
			mono: true,
			value: e.count,
			color: errorShade[Math.min(i, errorShade.length - 1)],
			cells: [String(e.count), percent(e.count, total)]
		}));
	});
	const kpiLoading = $derived(!summary && !$summaryError);
	const retrySummary = () => dashboard.fetchSummary(filters, { cache: 'no-store' }).catch(() => {});
</script>

<PageHeader title={l.translate('breadcrumb.overview')}>
	{#snippet actions()}
		<span
			class="mr-1 flex items-center gap-2 text-label {$liveStatus === 'offline'
				? 'text-warning'
				: 'text-muted-foreground'}"
			aria-live="polite"
		>
			{#if $liveStatus === 'live'}
				<span class="size-2 rounded-full bg-success ring-3 ring-success-soft"></span>
				{l.translate('dashboard.live')}
			{:else if $liveStatus === 'connecting'}
				<span class="size-2 rounded-full bg-faint"></span>
				{l.translate('dashboard.connecting')}
			{:else}
				<Icon icon={WifiDisconnected01Icon} class="size-3.5" />
				{l.translate('dashboard.offline')}
			{/if}
		</span>
		<span id="range-label" class="sr-only">{l.translate('dashboard.range.label')}</span>
		<Segmented
			labelledby="range-label"
			items={ranges}
			bind:value={() => String(range) as RangeKey, (v) => (range = Number(v) as DashboardRange)}
		/>
		<Select.Root
			type="single"
			value={environment}
			onValueChange={(value) => (environment = value as '' | Environment)}
		>
			<Select.Trigger class="w-44" aria-label={l.translate('dashboard.environment.label')}>
				{environments.find((e) => e.value === environment)?.label}
			</Select.Trigger>
			<Select.Content>
				{#each environments as e (e.value)}
					<Select.Item value={e.value} label={e.label}>
						<Icon
							icon={e.value === 'production'
								? Rocket01Icon
								: e.value === 'staging'
									? TestTube01Icon
									: Layers01Icon}
							class="text-muted-foreground"
						/>
						{e.label}
					</Select.Item>
				{/each}
			</Select.Content>
		</Select.Root>
		<NewAppMenu />
	{/snippet}
	{#snippet status()}
		{#if $apps}<span>{l.translate('dashboard.status.apps', [$apps.length])}</span>{/if}
		{#if $targets}<span>{l.translate('dashboard.status.targets', [activeTargets])}</span>{/if}
		{#if summary}
			<span
				>{l.translate('dashboard.status.deployments', [summary.current.deployments, range])}</span
			>
		{/if}
	{/snippet}
</PageHeader>

{#if firstRun}
	<section aria-labelledby="first-run" class="flex max-w-2xl flex-col gap-3">
		<Illustration name="rocket" class="mb-2 w-[200px] max-w-full" />
		<SectionHeading id="first-run">{l.translate('dashboard.first.title')}</SectionHeading>
		<p class="text-sm text-muted-foreground">{l.translate('dashboard.first.description')}</p>
		<ol class="flex flex-col gap-0.5 track p-2 sm:p-3">
			{#each [{ done: activeTargets > 0, label: l.translate('dashboard.first.target'), href: routes.createTarget }, { done: false, label: l.translate('dashboard.first.app'), href: routes.createApp }, { done: false, label: l.translate('dashboard.first.deploy') }] as step, i (i)}
				<li
					class="grid min-h-[54px] grid-cols-[28px_minmax(0,1fr)_auto] items-center gap-3 px-2 py-2"
				>
					<span
						class="flex size-7 items-center justify-center rounded-full text-label font-semibold {step.done
							? 'bg-success-soft text-success'
							: 'bg-raised text-muted-foreground'}"
					>
						{#if step.done}<Icon icon={Tick02Icon} class="size-3.5" />{:else}{i + 1}{/if}
					</span>
					<span class="text-sm {step.done ? 'text-muted-foreground line-through' : ''}"
						>{step.label}</span
					>
					{#if step.href && !step.done && (i === 0 || activeTargets > 0)}
						<Button size="sm" href={step.href}>
							<Icon icon={ArrowRight01Icon} />
							{l.translate('dashboard.first.go')}
						</Button>
					{/if}
				</li>
			{/each}
		</ol>
	</section>
{:else}
	<div class="flex flex-col gap-12">
		<StatsStrip {summary} {range} error={$summaryError} onretry={retrySummary} />

		<!-- The two charts, side by side from lg (owner, 2026-11-17). -->
		<DeployCharts
			{summary}
			{range}
			error={$summaryError}
			onretry={retrySummary}
			onfailures={() => (failuresOpen = true)}
			onrange={(r) => (range = r)}
		/>

		<div class="grid gap-x-10 gap-y-12 md:grid-cols-2 xl:grid-cols-3">
			<Tile
				title={l.translate('dashboard.environments')}
				icon={Layers01Icon}
				loading={kpiLoading}
				error={summary ? undefined : $summaryError}
				onretry={retrySummary}
			>
				{#snippet actions()}{l.translate('dashboard.env.note')}{/snippet}
				{#if environmentShare.length}
					<ShareBar
						items={environmentShare}
						headers={[
							l.translate('column.environment'),
							l.translate('dashboard.kpi.deployments'),
							l.translate('charts.share'),
							l.translate('dashboard.kpi.success')
						]}
						label={environmentShare.map((i) => `${i.label} ${i.value}`).join(', ')}
					/>
				{:else}
					{@render emptyRange()}
				{/if}
			</Tile>

			<Tile
				title={l.translate('dashboard.errors.title')}
				icon={CancelCircleIcon}
				loading={kpiLoading}
				error={summary ? undefined : $summaryError}
				onretry={retrySummary}
			>
				{#snippet actions()}
					{#if summary?.current.failed}{l.translate('dashboard.errors.note', [
							summary.current.failed
						])}{/if}
				{/snippet}
				{#if summary?.current.failed}
					<ShareBar
						items={errorShare}
						headers={[
							l.translate('dashboard.errors.code'),
							l.translate('dashboard.series.failed'),
							l.translate('charts.share')
						]}
						label={errorShare.map((i) => `${i.label} ${i.value}`).join(', ')}
					/>
					<Button
						variant="outline"
						size="sm"
						class="self-start"
						onclick={() => (failuresOpen = true)}
					>
						<Icon icon={ArrowRight01Icon} />
						{l.translate('dashboard.failures.see')}
					</Button>
				{:else}
					<EmptyState
						illustration="allClear"
						size="compact"
						level={3}
						class="flex-1 justify-center"
						title={l.translate('dashboard.failures.none.title')}
						description={l.translate('dashboard.failures.none')}
					/>
				{/if}
			</Tile>

			<Tile
				title={l.translate('dashboard.calendar')}
				icon={Calendar03Icon}
				class="md:col-span-2 xl:col-span-1"
				loading={!calendar && !$calendarError}
				error={calendar ? undefined : $calendarError}
				onretry={() =>
					dashboard.fetchSummary(calendarFilters, { cache: 'no-store' }).catch(() => {})}
			>
				{#snippet actions()}{l.translate('dashboard.calendar.note')}{/snippet}
				{#if calendar}
					<CalendarHeatmap
						days={calendar.series.map((d) => ({
							date: d.date,
							value: d.succeeded + d.failed + d.other
						}))}
						label={l.translate('dashboard.calendar.label')}
						count={(n) => l.translate('charts.deployments', [n])}
					/>
				{/if}
			</Tile>
		</div>

		<div class="grid gap-x-10 gap-y-12 md:grid-cols-2">
			<AttentionTile apps={$apps} targets={$targets} failingJobs={$failedJobs?.total} />
			<InfraTile
				targets={$targets}
				jobs={{
					running: $runningJobs?.total,
					pending: $pendingJobs?.total,
					failed: $failedJobs?.total
				}}
			/>
		</div>

		<div class="grid gap-x-10 gap-y-12 lg:grid-cols-2">
			<ActivityFeed
				environment={filters.environment}
				{refresh}
				live={$liveStatus === 'live'}
				{appName}
			/>
			<AppsTile {filters} {refresh} />
		</div>
	</div>
{/if}

{#snippet emptyRange()}
	<EmptyState
		illustration="chart"
		size="compact"
		level={3}
		class="flex-1 justify-center"
		title={l.translate('dashboard.empty.range.title', [range])}
	>
		{#snippet action()}
			{#if range !== 90}
				<Button variant="outline" size="sm" onclick={() => (range = range === 7 ? 30 : 90)}>
					{l.translate('dashboard.range', [range === 7 ? 30 : 90])}
				</Button>
			{/if}
		{/snippet}
	</EmptyState>
{/snippet}

<FailuresDrawer bind:open={failuresOpen} {filters} {appName} />
