<!--
	Custom domains of one environment (seelf#5; DESIGN.md §11.7 Custom domains), in a drawer opened
	from the settings page, like Variables. How it works first, then one section per service (its
	port, its default address, its domains with their DNS, CAA and certificate status and a hint),
	what the statuses mean, the DNS records to create, and a note on what saving does. Works on a
	copy: Cancel discards it, Save sends this environment's domains and the server redeploys it.
	Statuses come from GET /apps/{id}/domains/check, for the domains already saved.
-->
<script lang="ts">
	import Add01Icon from '@hugeicons/core-free-icons/Add01Icon';
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import Delete02Icon from '@hugeicons/core-free-icons/Delete02Icon';
	import RefreshIcon from '@hugeicons/core-free-icons/RefreshIcon';
	import Tick02Icon from '@hugeicons/core-free-icons/Tick02Icon';
	import Drawer from '#lib/components/drawer.svelte';
	import Icon from '#lib/components/icon.svelte';
	import Notice from '#lib/components/notice.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import { BadRequestError } from '#lib/error.ts';
	import {
		domainChips,
		isUnder,
		parseHostname,
		recordFor,
		unicodeOf,
		type ChipTone
	} from '#lib/hostname.ts';
	import l, { type AppTranslationsString } from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service, {
		type AppDetail,
		type DomainsCheck,
		type DomainStatus,
		type DomainsPerService
	} from '#lib/resources/apps.ts';
	import type { Environment } from '#lib/resources/deployments.ts';
	import { cn } from '#lib/utils.js';
	import CloudflareSetup from './cloudflare-setup.svelte';

	type Props = {
		open: boolean;
		app: AppDetail;
		environment: Environment;
		/** The environment's domains were saved (the app is refetched by the caller). */
		onsaved?: () => void;
	};

	let { open = $bindable(false), app, environment, onsaved }: Props = $props();

	const MAX_PER_SERVICE = 20;

	const config = $derived(app[environment]);
	const targetUrl = $derived(config.target.url);
	const targetHost = $derived.by(() => {
		try {
			return targetUrl ? new URL(targetUrl).hostname : undefined;
		} catch {
			return undefined;
		}
	});
	const latest = $derived(app.latest_deployments[environment]);

	// Services that can take a domain: those with a main http port in the latest deployment.
	const webServices = $derived(
		(latest?.state.services ?? [])
			.map((s) => ({
				name: s.name,
				entrypoint: s.entrypoints.find((e) => e.router === 'http' && !e.is_custom)
			}))
			.filter((s) => !!s.entrypoint)
	);

	let draft = $state<DomainsPerService>({});
	let drafts = $state<Record<string, string>>({}); // What is typed in each service's add field
	let typedErrors = $state<Record<string, string | undefined>>({});
	let serverErrors = $state<Record<string, { code: string; app?: string }>>({});
	let otherService = $state('');
	let removed = $state<string>();
	let check = $state<DomainsCheck>();
	let checking = $state(false);
	let saving = $state(false);
	let error = $state<string>();

	let cloudflare = $state<ReturnType<typeof CloudflareSetup>>();
	let cloudflareReady = $state(false);
	let settingUp = $state(false);
	let cloudflareVisible = $state(false);
	const atCloudflare = $derived((check?.domains ?? []).some((d) => d.cloudflare_dns));

	const statuses = $derived(new Map((check?.domains ?? []).map((d) => [d.hostname, d])));

	// A fresh copy, and fresh statuses, each time the drawer opens.
	$effect(() => {
		if (open) {
			draft = Object.fromEntries(
				Object.entries(config.domains ?? {}).map(([name, hostnames]) => [name, [...hostnames]])
			);
			drafts = {};
			typedErrors = {};
			serverErrors = {};
			removed = undefined;
			error = undefined;
			otherService = '';
			if (targetUrl) void runCheck();
		}
	});

	const sections = $derived.by(() => {
		const names = new Set([...webServices.map((s) => s.name), ...Object.keys(draft)]);
		return [...names].sort().map((name) => {
			const web = webServices.find((s) => s.name === name);
			const url = latest?.state.services?.find((s) => s.name === name)?.urls?.at(-1);
			return {
				name,
				port: web?.entrypoint?.port,
				defaultHost: url ? new URL(url).hostname : undefined,
				hostnames: draft[name] ?? []
			};
		});
	});

	async function runCheck() {
		checking = true;
		try {
			check = await service.checkDomains(app.id, environment);
		} catch {
			check = undefined;
		} finally {
			checking = false;
		}
	}

	function validate(serviceName: string, value: string): string | undefined {
		const parsed = parseHostname(value);
		if ('error' in parsed) return parsed.error;
		if (isUnder(parsed.value, targetHost)) return 'domain_reserved_by_target';
		const all = Object.values(draft).flat();
		if (all.includes(parsed.value)) return 'domain_duplicated';
		if ((draft[serviceName]?.length ?? 0) >= MAX_PER_SERVICE) return 'too_many_domains';
		return undefined;
	}

	function add(serviceName: string) {
		const value = drafts[serviceName] ?? '';
		const problem = validate(serviceName, value);
		typedErrors[serviceName] = problem;
		if (problem) return;
		const parsed = parseHostname(value) as { value: string };
		draft[serviceName] = [...(draft[serviceName] ?? []), parsed.value];
		drafts[serviceName] = '';
	}

	function remove(serviceName: string, hostname: string) {
		draft[serviceName] = (draft[serviceName] ?? []).filter((h) => h !== hostname);
		removed = hostname;
		serverErrors = {};
	}

	function addService() {
		const name = otherService.trim();
		if (!name || draft[name]) return;
		draft[name] = [];
		otherService = '';
	}

	/** Saves the domains, then sets them up on Cloudflare, keeping the drawer open for the tasks. */
	async function saveAndSetUp() {
		settingUp = true;
		try {
			if (!(await save(true))) return;
			await cloudflare?.setup();
			void runCheck();
		} finally {
			settingUp = false;
		}
	}

	async function save(keepOpen = false): Promise<boolean> {
		saving = true;
		error = undefined;
		serverErrors = {};
		const domains = Object.fromEntries(Object.entries(draft).filter(([, h]) => h.length > 0));
		const only = { target: config.target.id, vars: config.vars, domains };
		try {
			await service.update(app.id, {
				version_control: undefined,
				production: environment === 'production' ? only : undefined,
				staging: environment === 'staging' ? only : undefined
			});
			if (!keepOpen) open = false;
			onsaved?.();
			return true;
		} catch (ex) {
			if (ex instanceof BadRequestError && ex.isValidationError) {
				const prefix = `${environment}.domains`;
				const mapped: Record<string, { code: string; app?: string }> = {};
				for (const [key, detail] of Object.entries(ex.details)) {
					if (!key.startsWith(prefix)) continue;
					const [, serviceName, index] = key.slice(prefix.length).split('.');
					const hostname =
						serviceName !== undefined && index !== undefined
							? domains[serviceName]?.[Number(index)]
							: undefined;
					mapped[hostname ?? (serviceName || '*')] = {
						code: detail.code,
						app: (detail.detail as { app?: string } | undefined)?.app
					};
				}
				serverErrors = mapped;
				error = l.translate('domains.save.failed', [Object.keys(mapped).length]);
			} else {
				error =
					ex instanceof Error
						? l.translate(ex.message as AppTranslationsString)
						: l.translate('unexpected_error');
			}
			return false;
		} finally {
			saving = false;
		}
	}

	const chipFill: Record<ChipTone, string> = {
		success: 'bg-success-soft',
		warning: 'bg-warning-soft',
		danger: 'bg-danger-soft',
		muted: 'bg-raised'
	};
	const chipDot: Record<ChipTone, string> = {
		success: 'bg-success',
		warning: 'bg-warning',
		danger: 'bg-danger',
		muted: 'bg-faint'
	};

	/** One sentence on why a domain is in its state and what to do. */
	function hint(status: DomainStatus | undefined, primary: boolean): string {
		if (!status) return l.translate('domains.hint.unchecked');
		if (status.caa === 'caa_blocks')
			return l.translate('domains.hint.caa_blocks', [(status.caa_issuers ?? []).join(', ')]);
		switch (status.dns) {
			case 'no_record':
				return l.translate('domains.hint.no_record');
			case 'points_elsewhere':
				return l.translate('domains.hint.points_elsewhere', [
					status.addresses.join(', '),
					(check?.expected.addresses ?? []).join(', ')
				]);
			case 'behind_cloudflare':
				return l.translate('domains.hint.behind_cloudflare');
			case 'unknown':
				return l.translate('domains.hint.unknown');
		}
		if (check?.https && status.certificate === 'default_certificate')
			return l.translate('domains.hint.default_certificate');
		if (check?.https && status.certificate === 'unreachable')
			return l.translate('domains.hint.unreachable');
		return primary ? l.translate('domains.hint.primary') : l.translate('domains.hint.ready');
	}

	const records = $derived(
		sections.flatMap((s) =>
			s.hostnames.map((h) =>
				recordFor(h, s.defaultHost ?? check?.expected.host ?? '', check?.expected.addresses ?? [])
			)
		)
	);
	const mainService = $derived(webServices[0]?.name ?? sections[0]?.name);
</script>

<Drawer
	bind:open
	size="wide"
	title={l.translate('domains.title', [l.translate(`deploy.environment.${environment}`)])}
	description={l.translate('domains.description')}
>
	{#if !targetUrl}
		<Notice>
			{l.translate('domains.manual', [config.target.name ?? ''])}
			<a href={routes.editTarget(config.target.id)} class="font-semibold text-primary underline">
				{l.translate('domains.manual.link')}
			</a>
		</Notice>
		<p class="text-xs text-muted-foreground">{l.translate('domains.manual.why')}</p>
	{:else}
		{#if error}
			<Alert.Root variant="destructive">
				<Icon icon={Alert02Icon} />
				<Alert.Title class="font-semibold">{error}</Alert.Title>
				{#if serverErrors['*']}
					<Alert.Description
						>{l.translate(serverErrors['*'].code as AppTranslationsString)}</Alert.Description
					>
				{/if}
			</Alert.Root>
		{/if}

		{#if removed}
			<Notice tone="warning" role="status">
				{l.translate('domains.removed', [unicodeOf(removed)])}
			</Notice>
		{/if}

		<section class="flex flex-col gap-2.5" aria-labelledby="domains-how">
			<h3 id="domains-how" class="text-sm font-semibold">{l.translate('domains.how')}</h3>
			<ol class="grid gap-3 track p-4 text-xs text-muted-foreground sm:grid-cols-3">
				<li class="flex flex-col gap-1">
					<b class="text-label font-semibold text-ink">{l.translate('domains.how.1.title')}</b>
					{l.translate('domains.how.1')}
				</li>
				<li class="flex flex-col gap-1">
					<b class="text-label font-semibold text-ink">{l.translate('domains.how.2.title')}</b>
					{l.translate('domains.how.2')}
				</li>
				<li class="flex flex-col gap-1">
					<b class="text-label font-semibold text-ink">{l.translate('domains.how.3.title')}</b>
					{l.translate('domains.how.3', [l.translate(`deploy.environment.${environment}`)])}
				</li>
			</ol>
		</section>

		{#if webServices.length === 0}
			<Notice>{l.translate('domains.never_deployed')}</Notice>
		{/if}

		{#each sections as section (section.name)}
			<section class="flex flex-col gap-2" aria-labelledby="domains-{section.name}">
				<h3
					id="domains-{section.name}"
					class="flex flex-wrap items-baseline gap-x-2 text-sm font-semibold"
				>
					{section.name}
					<span class="text-xs font-normal text-muted-foreground">
						{#if section.port}{l.translate('domains.service.port', [section.port])}{/if}
						{#if section.defaultHost}
							· {l.translate('domains.service.also_at')}
							<span class="font-mono">{section.defaultHost}</span>
						{/if}
					</span>
				</h3>
				<p class="text-xs text-muted-foreground">
					{l.translate(
						section.name === mainService ? 'domains.service.main' : 'domains.service.other'
					)}
				</p>

				<div class="flex min-w-0 flex-col gap-2 track p-3 sm:p-4">
					<ul class="flex flex-col">
						{#each section.hostnames as hostname, index (hostname)}
							{@const status = statuses.get(hostname)}
							{@const problem = serverErrors[hostname]}
							<li
								class="grid grid-cols-[minmax(0,1fr)_auto] gap-x-2 gap-y-1.5 border-b border-line py-2.5"
							>
								<span class="flex min-w-0 flex-col">
									<span class="font-mono text-label font-semibold break-all">{hostname}</span>
									{#if unicodeOf(hostname) !== hostname}
										<span class="text-xs text-muted-foreground">{unicodeOf(hostname)}</span>
									{/if}
								</span>
								<Button
									variant="ghost"
									size="icon"
									aria-label={l.translate('domains.remove', [hostname])}
									onclick={() => remove(section.name, hostname)}
								>
									<Icon icon={Delete02Icon} />
								</Button>
								{#if problem}
									<p class="col-span-2 text-xs text-danger" role="alert">
										{problem.code === 'domain_already_used' && problem.app
											? l.translate('domains.error.used_by', [problem.app])
											: l.translate(problem.code as AppTranslationsString)}
									</p>
								{:else}
									<span class="col-span-2 flex flex-wrap gap-1.5">
										{#each domainChips(status, check?.https ?? false) as chip (chip.key)}
											<span
												class={cn(
													'inline-flex h-6 items-center gap-1.5 rounded-sm px-2 text-xs font-semibold text-ink',
													chipFill[chip.tone]
												)}
											>
												<span
													aria-hidden="true"
													class={cn('size-2 rounded-full', chipDot[chip.tone])}
												></span>
												{chip.key === 'domains.status.points_elsewhere'
													? l.translate('domains.status.points_elsewhere.at', [
															status?.addresses.join(', ') ?? ''
														])
													: l.translate(chip.key as AppTranslationsString)}
											</span>
										{/each}
									</span>
									<p class="col-span-2 text-xs text-muted-foreground">
										{hint(status, section.name === mainService && index === 0)}
									</p>
								{/if}
							</li>
						{/each}
					</ul>

					<Label for="domains-add-{section.name}" class="mt-1">
						{l.translate('domains.add', [section.name])}
					</Label>
					<p class="text-xs text-muted-foreground">{l.translate('domains.add.help')}</p>
					<form
						class="flex gap-2"
						onsubmit={(event) => {
							event.preventDefault();
							add(section.name);
						}}
					>
						<Input
							id="domains-add-{section.name}"
							bind:value={drafts[section.name]}
							oninput={() => (typedErrors[section.name] = undefined)}
							placeholder={l.translate('domains.add.placeholder')}
							class="font-mono text-label"
							autocomplete="off"
							spellcheck="false"
							aria-invalid={!!typedErrors[section.name] || undefined}
							aria-describedby="domains-add-{section.name}-error"
						/>
						<Button type="submit" variant="outline">
							<Icon icon={Add01Icon} />
							{l.translate('domains.add.button')}
						</Button>
					</form>
					{#if typedErrors[section.name]}
						<p id="domains-add-{section.name}-error" class="text-xs text-danger" role="alert">
							{l.translate(typedErrors[section.name] as AppTranslationsString)}
						</p>
					{/if}
				</div>
			</section>
		{/each}

		<form
			class="flex flex-col gap-1.5"
			onsubmit={(event) => {
				event.preventDefault();
				addService();
			}}
		>
			<Label for="domains-other-service">{l.translate('domains.other_service')}</Label>
			<p class="text-xs text-muted-foreground">{l.translate('domains.other_service.help')}</p>
			<div class="flex gap-2">
				<Input
					id="domains-other-service"
					bind:value={otherService}
					placeholder="api"
					autocomplete="off"
				/>
				<Button type="submit" variant="outline"
					>{l.translate('domains.other_service.button')}</Button
				>
			</div>
		</form>

		<CloudflareSetup
			bind:this={cloudflare}
			bind:ready={cloudflareReady}
			bind:visible={cloudflareVisible}
			targetId={config.target.id}
			targetName={config.target.name ?? ''}
			https={targetUrl.startsWith('https:')}
			{atCloudflare}
		/>

		<section class="flex flex-col gap-2" aria-labelledby="domains-legend">
			<h3 id="domains-legend" class="text-sm font-semibold">{l.translate('domains.legend')}</h3>
			<p class="text-xs text-muted-foreground">{l.translate('domains.legend.intro')}</p>
			<dl class="grid grid-cols-[140px_minmax(0,1fr)] gap-x-3 gap-y-1.5 track p-4 text-xs">
				{#each ['points_here', 'points_elsewhere', 'no_record', 'behind_cloudflare', 'caa_blocks', 'default_certificate', 'certificate_ok'] as const as key (key)}
					<dt class="font-semibold">{l.translate(`domains.status.${key}`)}</dt>
					<dd class="text-muted-foreground">{l.translate(`domains.legend.${key}`)}</dd>
				{/each}
			</dl>
		</section>

		{#if records.length && !(cloudflareVisible && cloudflareReady)}
			<section class="flex flex-col gap-2" aria-labelledby="domains-records">
				<h3 id="domains-records" class="text-sm font-semibold">{l.translate('domains.records')}</h3>
				<p class="text-xs text-muted-foreground">{l.translate('domains.records.help')}</p>
				<div class="overflow-x-auto track p-3 sm:p-4">
					<table class="w-full text-left text-xs">
						<thead class="text-muted-foreground">
							<tr>
								<th scope="col" class="py-1.5 pr-3 font-semibold"
									>{l.translate('domains.records.domain')}</th
								>
								<th scope="col" class="py-1.5 pr-3 font-semibold"
									>{l.translate('domains.records.type')}</th
								>
								<th scope="col" class="py-1.5 font-semibold"
									>{l.translate('domains.records.value')}</th
								>
							</tr>
						</thead>
						<tbody class="font-mono">
							{#each records as record (record.hostname)}
								<tr class="align-top">
									<td class="py-1.5 pr-3 break-all">{record.hostname}</td>
									<td class="py-1.5 pr-3 whitespace-nowrap">{record.type}</td>
									<td class="py-1.5 break-all">{record.values.join('\n')}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</section>
		{/if}

		<Notice
			>{l.translate('domains.notice', [l.translate(`deploy.environment.${environment}`)])}</Notice
		>
	{/if}

	{#snippet footer()}
		{#if targetUrl}
			<Button variant="outline" class="mr-auto" loading={checking} onclick={runCheck}>
				<Icon icon={RefreshIcon} />
				{l.translate('domains.check')}
			</Button>
		{/if}
		<Button variant="outline" onclick={() => (open = false)}>{l.translate('cancel')}</Button>
		{#if targetUrl && cloudflareVisible && cloudflareReady}
			<Button loading={settingUp} onclick={saveAndSetUp}>
				<Icon icon={Tick02Icon} />
				{l.translate('cloudflare.setup')}
			</Button>
		{:else if targetUrl}
			<Button loading={saving} onclick={() => save()}>
				<Icon icon={Tick02Icon} />
				{l.translate('domains.save')}
			</Button>
		{/if}
	{/snippet}
</Drawer>
