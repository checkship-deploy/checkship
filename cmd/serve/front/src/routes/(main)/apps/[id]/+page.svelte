<!--
	Application overview (DESIGN.md §11.7): this application's four figures (the dashboard summary
	filtered by app), staging and production as a pipeline side by side with Promote between them
	and each one's last 20 deploys, the two charts, then the services of both environments and the
	recent deployments. A running deployment gets a banner. Logs and previews
	open in drawers (app-workspace.svelte). Polls the application and the recent list.
-->
<script lang="ts">
	import ArrowDown02Icon from '@hugeicons/core-free-icons/ArrowDown02Icon';
	import ArrowRight01Icon from '@hugeicons/core-free-icons/ArrowRight01Icon';
	import CommandLineIcon from '@hugeicons/core-free-icons/CommandLineIcon';
	import ComputerTerminal01Icon from '@hugeicons/core-free-icons/ComputerTerminal01Icon';
	import ContainerIcon from '@hugeicons/core-free-icons/ContainerIcon';
	import LinkSquare02Icon from '@hugeicons/core-free-icons/LinkSquare02Icon';
	import Loading03Icon from '@hugeicons/core-free-icons/Loading03Icon';
	import RepeatIcon from '@hugeicons/core-free-icons/RepeatIcon';
	import Rocket01Icon from '@hugeicons/core-free-icons/Rocket01Icon';
	import ServerStack01Icon from '@hugeicons/core-free-icons/ServerStack01Icon';
	import Copy01Icon from '@hugeicons/core-free-icons/Copy01Icon';
	import { toast } from 'svelte-sonner';
	import TimelineGroup from '#lib/components/timeline/timeline-group.svelte';
	import TimelineList from '#lib/components/timeline/timeline-list.svelte';
	import TimelineRow from '#lib/components/timeline/timeline-row.svelte';
	import {
		clock,
		deploymentCommit,
		deploymentLabel,
		deploymentSource,
		deploymentTile
	} from '#lib/deployment-row.ts';
	import { groupByDay } from '#lib/timeline.ts';
	import { copyText } from '#lib/clipboard.ts';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import EnvironmentChip from '#lib/components/environment-chip.svelte';
	import RowMenuGroup from '#lib/components/table/row-menu-group.svelte';
	import RowMenuItem from '#lib/components/table/row-menu-item.svelte';
	import RowMenu from '#lib/components/table/row-menu.svelte';
	import Icon from '#lib/components/icon.svelte';
	import SectionHeading from '#lib/components/section-heading.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import appsService from '#lib/resources/apps.ts';
	import service, {
		DeploymentStatus,
		type Deployment,
		type DeploymentDetail,
		type Environment
	} from '#lib/resources/deployments.ts';
	import { deploymentBadge, deploymentMoment, isCancelled } from '#lib/status.ts';
	import { cn } from '#lib/utils.js';
	import DeployCharts from '#lib/components/dashboard/deploy-charts.svelte';
	import StatsStrip from '#lib/components/dashboard/stats-strip.svelte';
	import dashboard from '#lib/resources/dashboard.ts';
	import AppWorkspace, { type AppActions } from './app-workspace.svelte';
	import DeployStrip from '#lib/components/deploy-strip.svelte';
	import IconButton from '#lib/components/icon-button.svelte';
	import { liveUrls, socketAddresses } from './app-state.ts';
	import EmptyState from '#lib/components/empty-state.svelte';

	let { data } = $props();

	const appQuery = $derived(appsService.queryById(data.app.id));
	const appStore = $derived(appQuery.data);
	const app = $derived($appStore ?? data.app);

	// This application's figures and charts: the dashboard summary filtered by app (DESIGN.md §11.7).
	const summaryQuery = $derived(dashboard.querySummary({ range: 30, app: data.app.id }));
	const summaryStore = $derived(summaryQuery.data);
	const summaryError = $derived(summaryQuery.error);
	const retrySummary = () =>
		dashboard.fetchSummary({ range: 30, app: data.app.id }, { cache: 'no-store' }).catch(() => {});

	const recentQuery = $derived(service.queryAllByApp(data.app.id, { page: 1 }));
	const recentStore = $derived(recentQuery.data);
	const recent = $derived($recentStore?.data ?? []);

	const deleting = $derived(!!app.cleanup_requested_at);
	const staging = $derived(app.latest_deployments.staging);
	const production = $derived(app.latest_deployments.production);
	const running = $derived(
		(['staging', 'production'] as const)
			.map((environment) => app.latest_deployments[environment])
			.find((d) => d && !d.state.finished_at)
	);

	const isGit = (d?: Deployment) => d?.source.discriminator === 'git';
	const hashOf = (d?: Deployment) =>
		d?.source.discriminator === 'git' ? d.source.data.hash : undefined;

	// Promote is offered once staging has a successful deployment, and not while it runs.
	const canPromote = $derived(!deleting && staging?.state.status === DeploymentStatus.Succeeded);
	const newer = $derived(!!staging && isGit(staging) && hashOf(staging) !== hashOf(production));

	let promoteOpen = $state(false);
	let redeployOpen = $state(false);
	let redeployNumber = $state(0);

	function sourceLabel(d: Deployment) {
		const kind = l.translate(`deployment.source.${d.source.discriminator}`);
		return d.source.discriminator === 'git' ? `${kind} · ${d.source.data.branch}` : kind;
	}

	function timing(d: Deployment) {
		const { started_at, finished_at } = d.state;
		if (started_at && finished_at) {
			return `${l.relative(finished_at)} · ${l.translate('preview.took', [l.duration(started_at, finished_at)])}`;
		}
		if (started_at) return l.translate('preview.started', [l.relative(started_at)]);
		return l.translate('preview.queued', [l.relative(d.requested_at)]);
	}

	type EnvService = { url?: string; image: string };
	type ServiceRow = { name: string; production?: EnvService; staging?: EnvService };
	const services = $derived.by(() => {
		const rows: Record<string, ServiceRow> = {};
		for (const environment of ['production', 'staging'] as const) {
			for (const s of app.latest_deployments[environment]?.state.services ?? []) {
				rows[s.name] ??= { name: s.name };
				rows[s.name][environment] = { url: s.entrypoints.find((e) => e.url)?.url, image: s.image };
			}
		}
		return Object.values(rows).sort((a, b) => a.name.localeCompare(b.name));
	});

	/** "Production https://… · Staging internal · nginx:1.27": where each environment answers. */
	function serviceDetails(s: ServiceRow): string[] {
		const where = (env: EnvService | undefined) =>
			env ? (env.url ?? l.translate('detail.services.internal')) : undefined;
		return [
			s.production && l.translate('detail.services.row.production', [where(s.production)!]),
			s.staging && l.translate('detail.services.row.staging', [where(s.staging)!]),
			(s.production ?? s.staging)?.image
		].filter((part): part is string => !!part);
	}

	async function copy(text: string) {
		if (await copyText(text)) toast.success(l.translate('copied'));
		else toast.error(l.translate('copy_blocked'));
	}

	const copyLink = (number: number) =>
		copy(new URL(routes.deployment(app.id, number), location.origin).href);

	function cardState(latest: DeploymentDetail) {
		if (!latest.state.finished_at) return 'deploying';
		return latest.state.status === DeploymentStatus.Succeeded ? 'live' : 'failed';
	}
</script>

<AppWorkspace {app} apiKey={data.user.api_key} tab="overview">
	{#snippet children(workspace: AppActions)}
		<div class="flex flex-col gap-12">
			{#if deleting}
				<Alert.Root variant="destructive">
					<Icon icon={Loading03Icon} class="animate-spin" />
					<Alert.Title class="font-semibold">{l.translate('apps.deletion_requested')}</Alert.Title>
					<Alert.Description>{l.translate('app.deleting.description')}</Alert.Description>
				</Alert.Root>
			{:else if running}
				<div
					role="status"
					class="grid grid-cols-[20px_minmax(0,1fr)] items-center gap-3 rounded-lg bg-warning-soft py-3 pr-3 pl-4 sm:grid-cols-[20px_minmax(0,1fr)_auto]"
				>
					<Icon icon={Loading03Icon} class="size-5 animate-spin text-warning" />
					<span class="flex min-w-0 flex-col gap-0.5">
						<strong class="text-sm font-semibold">
							{l.translate('detail.running.title', [
								running.deployment_number,
								running.environment
							])}
						</strong>
						<span class="text-label">
							{timing(running)}{#if running.environment === 'staging' && production}
								· {l.translate('detail.running.serving', [production.deployment_number])}{/if}
						</span>
					</span>
					<Button
						variant="outline"
						size="sm"
						class="max-sm:col-span-2"
						onclick={() => workspace.openPreview(running.deployment_number)}
					>
						<Icon icon={CommandLineIcon} />
						{l.translate('detail.follow_logs')}
					</Button>
				</div>
			{/if}

			<StatsStrip
				summary={$summaryStore}
				range={30}
				label={l.translate('detail.figures.label', [30])}
				error={$summaryError}
				onretry={retrySummary}
			/>

			<section class="flex flex-col gap-3" aria-labelledby="environments-title">
				<SectionHeading id="environments-title">
					{l.translate('detail.environments')}
					{#snippet aside()}{l.translate('detail.environments.help')}{/snippet}
				</SectionHeading>

				<!-- A pipeline in the direction code moves (owner, 2026-11-17): staging, Promote, production
				     side by side from xl, stacked below. One track (owner, 2026-11-21, canvas "checkship
				     environments pipeline", option C): a single rounded outline in line-strong on the page's
				     own background holds both lanes, the named exception to §1 Cardless. Nothing inside gets
				     a box; the state stays in the marker and badge. Promote is centred on the gap and on the
				     lanes' height. -->
				<div
					class="grid gap-2 track p-3 xl:grid-cols-[minmax(0,1fr)_190px_minmax(0,1fr)] xl:items-stretch"
				>
					{#each ['staging', 'connector', 'production'] as const as slot (slot)}
						{#if slot === 'connector'}
							<div
								class="flex flex-col items-center justify-center gap-2.5 px-2 py-3 text-center xl:py-0"
							>
								<span
									class={cn(
										'flex size-10 shrink-0 items-center justify-center rounded-full xl:-rotate-90',
										canPromote ? 'bg-accent-soft text-primary' : 'bg-raised text-muted-foreground'
									)}
								>
									<Icon icon={ArrowDown02Icon} class="size-4.5" />
								</span>
								{#if staging}
									<Button
										variant={canPromote && newer ? 'default' : 'outline'}
										size="sm"
										disabled={!canPromote}
										class="h-auto max-w-full py-1.5 whitespace-normal"
										onclick={() => (promoteOpen = true)}
									>
										<Icon icon={Rocket01Icon} />
										{l.translate('detail.promote.to_production', [staging.deployment_number])}
									</Button>
									<span class="max-w-[190px] text-xs text-muted-foreground">
										{#if !staging.state.finished_at}
											{l.translate('detail.promote.wait', [staging.deployment_number])}
										{:else if !canPromote}
											{l.translate('detail.promote.failed')}
										{:else if newer}
											{l.translate('detail.promote.newer', [staging.deployment_number])}
										{:else}
											{l.translate('detail.promote.same')}
										{/if}
									</span>
								{:else}
									<span class="max-w-[190px] text-xs text-muted-foreground">
										{l.translate('detail.promote.none')}
									</span>
								{/if}
							</div>
						{:else}
							{@const environment = slot as Environment}
							{@const latest = app.latest_deployments[environment]}
							{@const target = app[environment].target}
							{@const urls = liveUrls(latest)}
							{@const url = urls[0]}
							{@const state = latest ? cardState(latest) : undefined}
							<section
								aria-labelledby="card-{environment}"
								class="flex min-w-0 flex-col gap-4 p-3 sm:p-5"
							>
								<div class="flex min-w-0 flex-col gap-2.5">
									<div class="flex flex-wrap items-center gap-2.5">
										<h3 id="card-{environment}" class="text-lg/[26px] font-semibold">
											{l.translate(`deploy.environment.${environment}`)}
										</h3>
										{#if state}
											<span
												class={cn(
													'flex items-center gap-1.5 text-xs font-semibold',
													state === 'live' && 'text-success',
													state === 'deploying' && 'text-warning',
													state === 'failed' && 'text-danger'
												)}
											>
												{#if state === 'deploying'}
													<Icon icon={Loading03Icon} class="size-3 animate-spin" />
												{:else}
													<span class="size-2 rounded-full bg-current"></span>
												{/if}
												{l.translate(`detail.card.${state}`)}
											</span>
										{/if}
										<a
											href={routes.editTarget(target.id)}
											class="flex h-6 items-center gap-1.5 rounded-sm bg-raised px-2 text-xs font-medium text-ink outline-none hover:bg-hover focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset"
										>
											<Icon icon={ServerStack01Icon} class="size-3.5" />
											{l.translate('app.on_target', [target.name])}
										</a>
									</div>
									{#if url}
										<a
											href={url}
											target="_blank"
											rel="noopener noreferrer"
											class="flex max-w-full min-w-0 items-center gap-1.5 self-start rounded-sm font-mono text-label text-primary outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring"
										>
											<span class="truncate">{url}</span>
											<Icon icon={LinkSquare02Icon} class="size-3.5 shrink-0" />
										</a>
										{#if urls.length > 1}
											<p class="text-xs text-muted-foreground">
												{l.translate('domains.also_at')}
												{#each urls.slice(1) as other, i (other)}
													{#if i > 0},{/if}
													<a
														href={other}
														target="_blank"
														rel="noopener noreferrer"
														class="font-mono hover:underline">{other.replace(/^https?:\/\//, '')}</a
													>
												{/each}
											</p>
											<p class="text-xs text-faint">{l.translate('domains.primary.explain')}</p>
										{/if}
									{:else if socketAddresses(latest, app.name, environment).length > 0}
										<!-- A database or a broker: no web page, but these addresses. -->
										<dl class="flex min-w-0 flex-col gap-1 text-xs">
											{#each socketAddresses(latest, app.name, environment) as socket (socket.inside)}
												{#if socket.outside}
													<div class="flex min-w-0 items-center gap-1.5">
														<dt class="shrink-0 text-muted-foreground">
															{l.translate('detail.tcp.outside')}
														</dt>
														<dd class="truncate font-mono text-label text-ink">{socket.outside}</dd>
														<IconButton
															icon={Copy01Icon}
															iconClass="size-3.5"
															class="size-6"
															label={l.translate('detail.tcp.copy', [socket.outside])}
															onclick={() => copy(socket.outside!)}
														/>
													</div>
												{/if}
												<div class="flex min-w-0 items-center gap-1.5">
													<dt class="shrink-0 text-muted-foreground">
														{l.translate('detail.tcp.apps')}
													</dt>
													<dd class="truncate font-mono text-label text-ink">{socket.inside}</dd>
													<IconButton
														icon={Copy01Icon}
														iconClass="size-3.5"
														class="size-6"
														label={l.translate('detail.tcp.copy', [socket.inside])}
														onclick={() => copy(socket.inside)}
													/>
												</div>
											{/each}
										</dl>
									{:else if latest}
										<p class="truncate text-label text-muted-foreground">
											{l.translate('app.no_url')}
										</p>
									{/if}
									{#if latest}
										{@const badge = deploymentBadge(latest.state)}
										<div class="flex flex-wrap items-center gap-3">
											<StatusBadge
												state={badge.state}
												label="{badge.label} · #{latest.deployment_number}"
											/>
											<span class="text-xs whitespace-nowrap text-muted-foreground"
												>{timing(latest)}</span
											>
										</div>
									{/if}
								</div>

								{#if latest}
									<dl
										class="grid min-w-0 grid-cols-[96px_minmax(0,1fr)] gap-x-3 gap-y-2 text-label/5"
									>
										<dt class="text-muted-foreground">{l.translate('deployment.source')}</dt>
										<dd class="truncate">
											{sourceLabel(latest)}
											{#if hashOf(latest)}
												· <span class="font-mono">{hashOf(latest)?.slice(0, 7)}</span>
											{/if}
										</dd>
										{#if latest.state.error_code && !isCancelled(latest.state)}
											<dt class="text-muted-foreground first-letter:uppercase">
												{l.translate('deployment.error_code')}
											</dt>
											<dd class="truncate font-mono text-danger">{latest.state.error_code}</dd>
										{:else if latest.state.services?.length}
											<dt class="text-muted-foreground first-letter:uppercase">
												{l.translate('deployment.services')}
											</dt>
											<dd class="truncate">
												{latest.state.services.map((s) => s.name).join(' · ')}
											</dd>
										{/if}
										<dt class="text-muted-foreground">{l.translate('deployment.queued_by')}</dt>
										<dd class="truncate">{latest.requested_by.email}</dd>
									</dl>
									<DeployStrip
										appId={app.id}
										{environment}
										live={state === 'live' ? latest.deployment_number : undefined}
										onopen={(n) => workspace.openPreview(n)}
									/>
									<div class="mt-auto flex flex-wrap gap-1.5">
										<Button
											variant="outline"
											size="sm"
											onclick={() => workspace.openPreview(latest.deployment_number)}
										>
											<Icon icon={CommandLineIcon} />
											{state === 'deploying'
												? l.translate('detail.follow_logs')
												: l.translate('detail.logs')}
										</Button>
										{#if state === 'live'}
											<Button
												variant="outline"
												size="sm"
												href={routes.appLogs(app.id, environment)}
											>
												<Icon icon={ComputerTerminal01Icon} />
												{l.translate('detail.app_logs')}
											</Button>
										{/if}
										{#if state !== 'deploying' && !deleting}
											<Button
												variant="outline"
												size="sm"
												onclick={() => {
													redeployNumber = latest.deployment_number;
													redeployOpen = true;
												}}
											>
												<Icon icon={RepeatIcon} />
												{l.translate('deployment.redeploy')}
											</Button>
										{/if}
									</div>
								{:else}
									<p class="text-sm text-muted-foreground">
										{l.translate(`app.env.empty.${environment}`)}
									</p>
									<div class="mt-auto flex">
										{#if !deleting}
											<Button
												variant="outline"
												size="sm"
												onclick={() => workspace.quickDeploy(environment)}
											>
												<Icon icon={Rocket01Icon} />
												{l.translate('app.env.deploy', [environment])}
											</Button>
										{/if}
									</div>
								{/if}
							</section>
						{/if}
					{/each}
				</div>
			</section>

			<DeployCharts
				summary={$summaryStore}
				range={30}
				error={$summaryError}
				onretry={retrySummary}
				app={app.name}
			/>

			<div class="grid items-start gap-8 xl:grid-cols-[minmax(0,1.25fr)_minmax(0,1fr)]">
				<section class="flex min-w-0 flex-col gap-3" aria-labelledby="services-title">
					<SectionHeading id="services-title">
						{l.translate('detail.services')}
						{#snippet aside()}
							{#if production || staging}
								{l.translate('detail.services.from', [
									production ? `#${production.deployment_number}` : '—',
									staging ? `#${staging.deployment_number}` : '—'
								])}
							{/if}
						{/snippet}
					</SectionHeading>
					<div class="min-w-0 track p-2 sm:p-3">
						{#if services.length > 0}
							<TimelineList labelledby="services-title">
								{#each services as s (s.name)}
									{@const url = s.production?.url ?? s.staging?.url}
									<TimelineRow
										icon={ContainerIcon}
										label={[s.name, ...serviceDetails(s)].join(', ')}
										href={url}
										external
									>
										{#snippet title()}{s.name}{/snippet}
										{#snippet details()}{serviceDetails(s).join(' · ')}{/snippet}
										{#snippet actions()}
											<RowMenu label={l.translate('table.menu', [s.name])}>
												<RowMenuGroup>
													{#if s.production?.url}
														<RowMenuItem icon={LinkSquare02Icon} href={s.production.url} external>
															{l.translate('menu.open_production')}
														</RowMenuItem>
													{/if}
													{#if s.staging?.url}
														<RowMenuItem icon={LinkSquare02Icon} href={s.staging.url} external>
															{l.translate('menu.open_staging')}
														</RowMenuItem>
													{/if}
													{#if url}
														<RowMenuItem icon={Copy01Icon} onSelect={() => copy(url)}>
															{l.translate('menu.copy_url')}
														</RowMenuItem>
													{/if}
													<RowMenuItem icon={CommandLineIcon} href={routes.appLogs(app.id)}>
														{l.translate('menu.logs')}
													</RowMenuItem>
												</RowMenuGroup>
											</RowMenu>
										{/snippet}
									</TimelineRow>
								{/each}
							</TimelineList>
						{:else}
							<EmptyState
								illustration="rocket"
								size="compact"
								level={3}
								title={l.translate('detail.services.none.title')}
								description={l.translate('detail.services.none')}
							/>
						{/if}
					</div>
				</section>

				<section class="flex min-w-0 flex-col gap-3" aria-labelledby="recent-title">
					<SectionHeading id="recent-title">
						{l.translate('detail.recent')}
						{#snippet aside()}
							<a
								href={routes.appDeployments(app.id)}
								class="flex items-center gap-1 rounded-sm text-label font-semibold text-primary outline-none hover:underline focus-visible:ring-2 focus-visible:ring-ring"
							>
								{l.translate('detail.recent.all')}
								<Icon icon={ArrowRight01Icon} class="size-3.5" />
							</a>
						{/snippet}
					</SectionHeading>
					<div class="min-w-0 track p-2 sm:p-3">
						{#if recent.length > 0}
							<div class="flex flex-col gap-3" aria-labelledby="recent-title" role="group">
								{#each groupByDay(recent, deploymentMoment, l.locale()) as group (group.key)}
									<TimelineGroup label={group.label} count={group.items.length} sticky={false}>
										{#each group.items as d (d.deployment_number)}
											{@const tile = deploymentTile(d.state)}
											<TimelineRow
												icon={tile.icon}
												tone={tile.tone}
												spin={tile.spin}
												label={deploymentLabel(d)}
												selected={workspace.previewNumber() === d.deployment_number}
												onclick={() => workspace.openPreview(d.deployment_number)}
											>
												{#snippet title()}#{d.deployment_number}{/snippet}
												{#snippet badge()}<EnvironmentChip environment={d.environment} />{/snippet}
												{#snippet details()}
													{clock(deploymentMoment(d))}
													{#if d.state.error_code && !isCancelled(d.state)}
														· <span class="font-mono text-danger">{d.state.error_code}</span>
													{:else if deploymentCommit(d.source)}
														· <span class="font-mono">{deploymentCommit(d.source)}</span>
													{:else if deploymentSource(d.source)}
														· {deploymentSource(d.source)}
													{/if}
												{/snippet}
												{#snippet actions()}
													<RowMenu label={l.translate('detail.row.menu', [d.deployment_number])}>
														<RowMenuGroup>
															<RowMenuItem
																icon={CommandLineIcon}
																onSelect={() => workspace.openPreview(d.deployment_number)}
															>
																{l.translate('detail.row.logs')}
															</RowMenuItem>
															{#if !deleting && d.state.finished_at}
																<RowMenuItem
																	icon={RepeatIcon}
																	onSelect={() => {
																		redeployNumber = d.deployment_number;
																		redeployOpen = true;
																	}}
																>
																	{l.translate('deployment.redeploy')}
																</RowMenuItem>
															{/if}
															<RowMenuItem
																icon={Copy01Icon}
																onSelect={() => copyLink(d.deployment_number)}
															>
																{l.translate('detail.row.copy')}
															</RowMenuItem>
														</RowMenuGroup>
													</RowMenu>
												{/snippet}
											</TimelineRow>
										{/each}
									</TimelineGroup>
								{/each}
							</div>
						{:else}
							<EmptyState
								illustration="rocket"
								size="compact"
								level={3}
								title={l.translate('app.empty.title')}
								description={l.translate('app.empty.description')}
							>
								{#snippet action()}
									{#if !deleting}
										<Button size="sm" onclick={() => workspace.openDeploy('production')}>
											<Icon icon={Rocket01Icon} />
											{l.translate('deployment.new')}
										</Button>
									{/if}
								{/snippet}
							</EmptyState>
						{/if}
					</div>
				</section>
			</div>
		</div>

		{#if staging}
			<ConfirmDialog
				bind:open={promoteOpen}
				title={l.translate('deployment.promote.title')}
				description={l.translate('deployment.promote.description', [staging.deployment_number])}
				confirmLabel={l.translate('deployment.promote.title')}
				icon={Rocket01Icon}
				onconfirm={() =>
					service
						.promote(app.id, staging.deployment_number)
						.then((d) => workspace.openPreview(d.deployment_number))}
			/>
		{/if}
		<ConfirmDialog
			bind:open={redeployOpen}
			title={l.translate('deployment.redeploy')}
			description={l.translate('deployment.redeploy.description', [
				redeployNumber,
				redeployNumber === production?.deployment_number ? 'production' : 'staging'
			])}
			confirmLabel={l.translate('deployment.redeploy')}
			icon={RepeatIcon}
			onconfirm={() =>
				service
					.redeploy(app.id, redeployNumber)
					.then((d) => workspace.openPreview(d.deployment_number))}
		/>
	{/snippet}
</AppWorkspace>
