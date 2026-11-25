<!--
	Sign-in and connection (DESIGN.md §11.7): everything a deployed application asks for when you
	open it, and where your other apps reach it, in one place. Opened from the application's header,
	from its row menu on Applications, and with ?access on its URL.

	For an app recognised as a template (lib/templates/recognise.ts), the values come from its own
	variables, read through the template's credentials and connection string, so they are what the
	app was deployed with, even after an edit. Any app also lists its tcp and udp addresses: from
	outside (the port the target published) and from other apps (the service's alias).
-->
<script lang="ts">
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import Settings02Icon from '@hugeicons/core-free-icons/Settings02Icon';
	import { appUrl } from '#lib/app-url.ts';
	import Drawer from '#lib/components/drawer.svelte';
	import Icon from '#lib/components/icon.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import type { AppDetail } from '#lib/resources/apps.ts';
	import type { Environment } from '#lib/resources/deployments.ts';
	import CredentialRows, { type CredentialRow } from '#lib/templates/credential-rows.svelte';
	import TemplateLogo from '#lib/templates/template-logo.svelte';
	import type { Template } from '#lib/templates/types.ts';
	import {
		connectionUrl,
		credentialValue,
		hasWebPage,
		internalHost,
		secretOf,
		valuesOf
	} from '#lib/templates/values.ts';
	import { cn } from '#lib/utils.js';
	import { liveUrl, socketAddresses } from './app-state.ts';

	type Props = { open: boolean; app: AppDetail; template?: Template };

	let { open = $bindable(false), app, template }: Props = $props();

	let environment = $state<Environment>('production');

	const latest = $derived(app.latest_deployments[environment]);
	const hasStaging = $derived(!!app.latest_deployments.staging);
	const values = $derived(template ? valuesOf(template, app[environment].vars) : {});
	const url = $derived(
		liveUrl(latest) ?? appUrl(app[environment].target.url, app.name, environment)
	);
	const webPage = $derived(!template || hasWebPage(template));

	/** What its page asks for: the template's credentials, from this environment's variables. */
	const signIn = $derived<CredentialRow[]>(
		template
			? (template.credentials ?? []).map((c) => {
					const value = credentialValue(template, c, values, url);
					const link = /^https?:\/\//.test(value);
					return {
						label: l.translate(c.label),
						value,
						secret: c.secret,
						href: link ? value : undefined
					};
				})
			: []
	);

	/** Where other apps reach it, and from outside the target. */
	const sockets = $derived(socketAddresses(latest, app.name, environment));
	const connect = $derived.by<CredentialRow[]>(() => {
		const rows: CredentialRow[] = [];
		if (template?.connect) {
			const host = internalHost(template, app.name, environment);
			rows.push({
				label: l.translate('templates.connect.host'),
				value: `${host}:${template.connect.port}`
			});
			const connection = connectionUrl(template, values, host);
			if (connection) {
				rows.push({
					label: l.translate('templates.connect.url'),
					value: connection,
					// A field placeholder other than {host} puts a secret in it: mask that part only.
					secret: /\{(?!host\})[A-Za-z0-9_]+\}/.test(template.connect.url ?? ''),
					secrets: template.fields
						.filter((f) => f.kind === 'generated' || f.secret)
						.map((f) => secretOf(f, values[f.key]))
				});
			}
		} else {
			for (const s of sockets) {
				rows.push({ label: `${l.translate('detail.tcp.apps')} · ${s.service}`, value: s.inside });
			}
		}
		for (const s of sockets) {
			if (!s.outside) continue;
			rows.push({
				label: `${l.translate('access.outside')} · ${s.service}`,
				value: s.outside,
				help: l.translate('access.outside.help')
			});
		}
		return rows;
	});
</script>

<Drawer
	bind:open
	title={l.translate('access.open')}
	description={l.translate('access.description', [app.name])}
>
	{#snippet leading()}
		{#if template}<TemplateLogo {template} class="size-10" />{/if}
	{/snippet}

	<div class="flex flex-col gap-6">
		{#if hasStaging}
			<div
				role="radiogroup"
				aria-label={l.translate('dashboard.environment.label')}
				class="flex gap-1 self-start rounded-md bg-sunken p-1"
			>
				{#each ['production', 'staging'] as const as env (env)}
					<button
						type="button"
						role="radio"
						aria-checked={environment === env}
						class={cn(
							'rounded-sm px-3 py-1 text-sm font-medium outline-none focus-visible:ring-2 focus-visible:ring-ring',
							environment === env ? 'bg-surface text-ink shadow-xs' : 'text-muted-foreground'
						)}
						onclick={() => (environment = env)}
					>
						{l.translate(`deploy.environment.${env}`)}
					</button>
				{/each}
			</div>
		{/if}

		{#if !latest}
			<p class="text-sm text-muted-foreground">{l.translate('access.not_deployed')}</p>
		{/if}

		{#if template && webPage}
			<section aria-labelledby="access-open" class="flex flex-col gap-2">
				<h3 id="access-open" class="text-sm font-semibold">{l.translate('access.open_it')}</h3>
				{#if url}
					<CredentialRows
						rows={[{ label: l.translate('detail.visit'), value: url, href: url }, ...signIn]}
					/>
				{:else if signIn.length > 0}
					<CredentialRows rows={signIn} />
				{/if}
				{#if signIn.length === 0}
					<p class="text-sm text-muted-foreground">
						{l.translate('access.no_signin')}
						{l.translate(template.after)}
					</p>
				{/if}
			</section>
		{:else if template && signIn.length > 0}
			<section aria-labelledby="access-signin" class="flex flex-col gap-2">
				<h3 id="access-signin" class="text-sm font-semibold">{l.translate('access.signin')}</h3>
				<CredentialRows rows={signIn} />
			</section>
		{/if}

		{#if connect.length > 0}
			<section aria-labelledby="access-connect" class="flex flex-col gap-2">
				<h3 id="access-connect" class="text-sm font-semibold">
					{l.translate('templates.connect.title')}
				</h3>
				<p class="text-sm text-muted-foreground">
					{l.translate('templates.connect.description', [app[environment].target.name])}
				</p>
				<CredentialRows rows={connect} />
			</section>
		{/if}

		{#if template?.firstStartOnly && signIn.length > 0}
			<Alert.Root>
				<Icon icon={Alert02Icon} />
				<Alert.Description>{l.translate('access.first_start', [template.name])}</Alert.Description>
			</Alert.Root>
		{/if}

		{#if template && (signIn.length > 0 || !webPage)}
			<div class="flex flex-col">
				<span class="text-sm font-medium">{l.translate('templates.after')}</span>
				<p class="text-sm text-muted-foreground">{l.translate(template.after)}</p>
			</div>
		{/if}

		{#if !template}
			<p class="text-sm text-muted-foreground">{l.translate('access.no_template')}</p>
		{/if}
	</div>

	{#snippet footer()}
		<Button variant="outline" href={routes.editApp(app.id)}>
			<Icon icon={Settings02Icon} />
			{l.translate('access.variables')}
		</Button>
		<Button onclick={() => (open = false)}>{l.translate('close')}</Button>
	{/snippet}
</Drawer>
