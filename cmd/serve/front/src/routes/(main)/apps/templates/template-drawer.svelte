<!--
	Review and configure a template (DESIGN.md §11.6 Drawers, §11.7 Templates gallery). The name and
	target, the URL it will get, then the template's settings in three groups: the ones you must
	fill (Deploy stays off until they are set), the ones filled in for you (generated secrets and
	values derived from the URL, all saved as app variables), and optional switches. Deploy creates
	the application with those variables on both environments, then queues its first production
	deployment from the template's compose file, and opens that deployment.
-->
<script lang="ts">
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import Copy01Icon from '@hugeicons/core-free-icons/Copy01Icon';
	import RefreshIcon from '@hugeicons/core-free-icons/RefreshIcon';
	import Rocket01Icon from '@hugeicons/core-free-icons/Rocket01Icon';
	import ServerStack01Icon from '@hugeicons/core-free-icons/ServerStack01Icon';
	import { toast } from 'svelte-sonner';
	import { goto } from '$app/navigation';
	import { appUrl, isSecureContextUrl } from '#lib/app-url.ts';
	import { copyText } from '#lib/clipboard.ts';
	import Drawer from '#lib/components/drawer.svelte';
	import FieldMessage from '#lib/components/field-message.svelte';
	import Icon from '#lib/components/icon.svelte';
	import IconButton from '#lib/components/icon-button.svelte';
	import SettingRow from '#lib/components/setting-row.svelte';
	import * as Alert from '#lib/components/ui/alert/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import { Switch } from '#lib/components/ui/switch/index.js';
	import { GLOBAL_ERROR_NAME, submitter } from '#lib/form.ts';
	import l, { type AppTranslationsString } from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import appsService, { type App } from '#lib/resources/apps.ts';
	import deploymentsService from '#lib/resources/deployments.ts';
	import type { Target } from '#lib/resources/targets.ts';
	import CredentialRows from '#lib/templates/credential-rows.svelte';
	import TemplateLogo from '#lib/templates/template-logo.svelte';
	import type { Template, TemplateField } from '#lib/templates/types.ts';
	import {
		blockingFields,
		connectionUrl,
		credentialValue,
		derivedValue,
		fieldProblem,
		freeName,
		generateSecret,
		hasWebPage,
		initialValues,
		inputFields,
		internalHost,
		newSecret,
		resolveVars,
		secretOf,
		type TemplateValues
	} from '#lib/templates/values.ts';
	import { cn } from '#lib/utils.js';

	type Props = {
		open: boolean;
		template: Template;
		targets: Target[];
		/** Existing applications, to say a name is taken before the server does. */
		apps: App[];
	};

	let { open = $bindable(false), template, targets, apps }: Props = $props();

	const formId = 'template-form';
	const appNamePattern = /^[a-z0-9-_]+$/;

	// The drawer is keyed by template, so these start fresh for each one.
	// svelte-ignore state_referenced_locally
	let target = $state(targets.find((t) => t.state.status === 2)?.id ?? targets[0]?.id ?? '');
	/** Names already used on a target: an app's name must be unique there. */
	const namesOn = (id: string) =>
		apps
			.filter((a) => a.production_target.id === id || a.staging_target.id === id)
			.map((a) => a.name);
	// Deployed once already? Suggest the next free name rather than open on an error.
	// svelte-ignore state_referenced_locally
	let name = $state(freeName(template.id, namesOn(target)));
	// svelte-ignore state_referenced_locally
	let values = $state<TemplateValues>(initialValues(template));
	let touched = $state<Record<string, boolean>>({});
	let tried = $state(false);

	const targetUrl = $derived(targets.find((t) => t.id === target)?.url);
	const url = $derived(appUrl(targetUrl, name.trim() || template.id));
	const inputs = $derived(inputFields(template, url));
	const generated = $derived(template.fields.filter((f) => f.kind === 'generated'));
	const fromUrl = $derived(url ? template.fields.filter((f) => f.kind === 'derived') : []);
	const optional = $derived(template.fields.filter((f) => f.kind === 'optional'));
	const blocking = $derived(blockingFields(template, values, url));
	const insecure = $derived(!!template.needsSecureContext && !!url && !isSecureContextUrl(url));
	/** Where the person's other apps reach the service, and the string they connect with. */
	const host = $derived(
		template.connect
			? `${internalHost(template, name.trim() || template.id)}:${template.connect.port}`
			: undefined
	);
	const connection = $derived(
		template.connect
			? connectionUrl(template, values, internalHost(template, name.trim() || template.id))
			: undefined
	);
	/** What its page asks for once deployed, from the values as they are now. */
	const signIn = $derived(
		(template.credentials ?? []).map((c) => ({
			label: l.translate(c.label),
			value: credentialValue(template, c, values, url),
			secret: c.secret
		}))
	);
	/** The connection string with its secrets masked, for display. */
	const shownConnection = $derived.by(() => {
		if (!connection) return undefined;
		let shown = connection;
		for (const field of template.fields.filter((f) => f.kind === 'generated')) {
			const secret = secretOf(field, values[field.key]);
			if (secret) shown = shown.replaceAll(secret, masked(secret));
		}
		return shown;
	});

	const targetLabel = (id: string) => {
		const t = targets.find((t) => t.id === id);
		return t ? `${t.name} · ${t.url ?? l.translate('target.manual_proxy')}` : '';
	};

	const { submit, loading, errors } = submitter(async () => {
		// Each environment gets values derived from its own address: staging lives at
		// <app>-staging.<host>, and a ROOT_URL pointing at production would send it there.
		const configFor = (environment: 'production' | 'staging') => {
			const vars = resolveVars(
				template,
				values,
				appUrl(targetUrl, name.trim() || template.id, environment)
			);
			return { target, vars: Object.keys(vars).length ? vars : undefined };
		};
		const app = await appsService.create({
			name: name.trim(),
			production: configFor('production'),
			staging: configFor('staging')
		});

		let number: number;
		try {
			const deployment = await deploymentsService.queue(app.id, {
				environment: 'production',
				raw: template.compose
			});
			number = deployment.deployment_number;
		} catch {
			// The application exists: open it rather than lose it, and say what is left to do.
			toast.error(l.translate('templates.queue_failed', [app.name]));
			open = false;
			await goto(routes.app(app.id));
			return;
		}

		open = false;
		await goto(routes.deployment(app.id, number));
	});

	const serverError = (key: string) => $errors?.[key] ?? undefined;
	const globalError = $derived($errors?.[GLOBAL_ERROR_NAME]);

	/** The name's problem: what the browser can tell first, then what the server said. */
	const nameError = $derived.by(() => {
		const n = name.trim();
		if (!appNamePattern.test(n) || n.endsWith('-staging')) return 'invalid_app_name';
		if (namesOn(target).includes(n)) return 'app_name_already_taken';
		const fromServer = [serverError('production.target'), serverError('staging.target')].find(
			(e) => e === 'app_name_already_taken'
		);
		return serverError('name') ?? fromServer;
	});

	const targetError = $derived(
		[serverError('production.target'), serverError('staging.target')].find(
			(e) => !!e && e !== 'app_name_already_taken'
		)
	);

	const reason = $derived.by(() => {
		if (nameError) return l.translate('templates.reason.name');
		const missing = blocking.filter((f) => !(values[f.key] ?? '').trim()).length;
		if (missing) return l.translate('templates.reason.fill', [missing]);
		if (blocking.length) return l.translate('templates.reason.fix');
		return undefined;
	});

	const shownProblem = (field: TemplateField) =>
		touched[field.key] || tried ? fieldProblem(field, values[field.key]) : undefined;

	function masked(value: string): string {
		return value.length > 12 ? `${value.slice(0, 6)}••••••••${value.slice(-4)}` : '••••••••';
	}

	async function copy(value: string) {
		if (await copyText(value)) toast.success(l.translate('copied'));
		else toast.error(l.translate('copy_blocked'));
	}

	function onsubmit(event: SubmitEvent) {
		event.preventDefault();
		if ($loading || targets.length === 0) return;
		tried = true;
		if (nameError) {
			document.getElementById('template-name')?.focus();
			return;
		}
		if (blocking.length) {
			document.getElementById(`template-${blocking[0].key}`)?.focus();
			return;
		}
		submit().catch(() => {});
	}
</script>

<Drawer bind:open title={l.translate('templates.deploy_title', [template.name])}>
	{#snippet leading()}
		<TemplateLogo {template} class="size-10" />
	{/snippet}
	{#snippet description()}
		{l.translate(`templates.category.${template.category}`)} ·
		<code class="font-mono text-label">{template.image}</code>
	{/snippet}

	<form id={formId} class="flex flex-col gap-6" novalidate {onsubmit}>
		<p class="text-sm text-muted-foreground">{l.translate(template.description)}</p>

		{#if globalError}
			<Alert.Root variant="destructive">
				<Icon icon={Alert02Icon} />
				<Alert.Title class="font-semibold">
					{l.translate(globalError as AppTranslationsString)}
				</Alert.Title>
			</Alert.Root>
		{/if}

		{#if targets.length === 0}
			<Alert.Root variant="warning">
				<Icon icon={Alert02Icon} />
				<Alert.Title class="font-semibold">{l.translate('app.no_targets')}</Alert.Title>
				<Alert.Description>
					<a href={routes.createTarget} class="font-semibold underline">
						{l.translate('settings.create_target')}
					</a>
				</Alert.Description>
			</Alert.Root>
		{/if}

		<fieldset class="flex flex-col gap-6" disabled={$loading}>
			<div class="flex flex-col gap-2">
				<Label for="template-name">{l.translate('name')} <span class="text-danger">*</span></Label>
				<Input
					id="template-name"
					bind:value={name}
					required
					autocomplete="off"
					spellcheck={false}
					aria-invalid={!!nameError || undefined}
					aria-describedby="template-name-message"
				/>
				<FieldMessage id="template-name-message" error={nameError}>
					{#snippet help()}{l.translate('settings.name.help')}{/snippet}
				</FieldMessage>
			</div>

			<div class="flex flex-col gap-2">
				<span id="template-target-label" class="text-sm font-medium">
					{l.translate('templates.target')}
				</span>
				<Select.Root type="single" bind:value={target}>
					<Select.Trigger
						aria-labelledby="template-target-label"
						aria-invalid={!!targetError || undefined}
						aria-describedby="template-target-message"
						class="w-full min-w-0"
					>
						<span class="truncate">{targetLabel(target)}</span>
					</Select.Trigger>
					<Select.Content>
						{#each targets as t (t.id)}
							<Select.Item value={t.id} label={t.name}>
								<Icon icon={ServerStack01Icon} class="text-muted-foreground" />
								{targetLabel(t.id)}
							</Select.Item>
						{/each}
					</Select.Content>
				</Select.Root>
				<FieldMessage id="template-target-message" error={targetError}>
					{#snippet help()}
						{#if url && !hasWebPage(template)}
							{l.translate('templates.no_page')}
						{:else if url}
							{l.translate('templates.live_at')}
							<code class="text-ink">{url}</code>
						{:else}
							{l.translate('templates.no_url')}
						{/if}
					{/snippet}
				</FieldMessage>
			</div>

			{#if insecure}
				<Alert.Root variant="warning">
					<Icon icon={Alert02Icon} />
					<Alert.Description
						>{l.translate('templates.https_needed', [template.name])}</Alert.Description
					>
				</Alert.Root>
			{/if}

			{#if inputs.length > 0}
				<section aria-labelledby="template-required" class="flex flex-col gap-4">
					<div class="flex flex-col">
						<h3 id="template-required" class="text-sm font-semibold">
							{l.translate('templates.required.title')}
						</h3>
						<p class="text-sm text-muted-foreground">
							{l.translate('templates.required.description', [template.name])}
						</p>
					</div>
					{#each inputs as field (field.key)}
						{@const problem = shownProblem(field)}
						<div class="flex flex-col gap-2">
							<Label for="template-{field.key}">
								{l.translate(field.label)} <span class="text-danger">*</span>
							</Label>
							<div class="flex gap-2">
								<Input
									id="template-{field.key}"
									type={field.email ? 'email' : 'text'}
									bind:value={values[field.key]}
									onblur={() => (touched[field.key] = true)}
									required
									autocomplete="off"
									spellcheck={false}
									class={cn('min-w-0 flex-1', field.secret && 'font-mono')}
									aria-invalid={!!problem || undefined}
									aria-describedby="template-{field.key}-message"
								/>
								{#if field.secret}
									<Button
										variant="outline"
										onclick={() => {
											values[field.key] = generateSecret(field.length ?? 20);
											touched[field.key] = true;
										}}
									>
										<Icon icon={RefreshIcon} />
										{l.translate('templates.generate')}
									</Button>
								{/if}
							</div>
							<FieldMessage id="template-{field.key}-message" error={problem}>
								{#snippet help()}
									{#if field.help}{l.translate(field.help)}{/if}
								{/snippet}
							</FieldMessage>
						</div>
					{/each}
				</section>
			{/if}

			{#if generated.length > 0 || fromUrl.length > 0}
				<section aria-labelledby="template-auto" class="flex flex-col gap-2">
					<div class="mb-1 flex flex-col">
						<h3 id="template-auto" class="text-sm font-semibold">
							{l.translate('templates.auto.title')}
						</h3>
						<p class="text-sm text-muted-foreground">{l.translate('templates.auto.description')}</p>
					</div>
					{#each fromUrl as field (field.key)}
						<div class="flex flex-col rounded-md bg-sunken px-3 py-2.5">
							<span class="flex items-center gap-2">
								<span class="text-sm font-medium">{l.translate(field.label)}</span>
								<span class="text-xs text-muted-foreground"
									>{l.translate('templates.auto.from_url')}</span
								>
							</span>
							<code class="font-mono text-xs/5 break-all text-muted-foreground"
								>{field.key}={derivedValue(field, url)}</code
							>
						</div>
					{/each}
					{#each generated as field (field.key)}
						<div class="flex items-center gap-2 rounded-md bg-sunken px-3 py-2.5">
							<span class="flex min-w-0 flex-1 flex-col">
								<span class="flex items-center gap-2">
									<span class="text-sm font-medium">{l.translate(field.label)}</span>
									<span class="text-xs text-muted-foreground"
										>{l.translate('templates.auto.generated')}</span
									>
								</span>
								<code class="font-mono text-xs/5 break-all text-muted-foreground"
									>{field.key}={field.prefix ?? ''}{masked(
										secretOf(field, values[field.key])
									)}</code
								>
							</span>
							<IconButton
								type="button"
								icon={Copy01Icon}
								iconClass="size-4"
								class="size-8"
								label={l.translate('templates.copy', [l.translate(field.label)])}
								onclick={() => copy(secretOf(field, values[field.key]))}
							/>
							<IconButton
								type="button"
								icon={RefreshIcon}
								iconClass="size-4"
								class="size-8"
								label={l.translate('templates.regenerate', [l.translate(field.label)])}
								onclick={() => (values[field.key] = newSecret(field))}
							/>
						</div>
					{/each}
				</section>
			{/if}

			{#each optional as field (field.key)}
				<SettingRow
					for="template-{field.key}"
					label={l.translate(field.label)}
					description={field.help ? l.translate(field.help) : undefined}
					class="py-0"
				>
					<Switch
						id="template-{field.key}"
						checked={values[field.key] === 'true'}
						onCheckedChange={(checked: boolean) => (values[field.key] = String(checked))}
					/>
				</SettingRow>
			{/each}

			{#if host}
				<section aria-labelledby="template-connect" class="flex flex-col gap-2">
					<div class="mb-1 flex flex-col">
						<h3 id="template-connect" class="text-sm font-semibold">
							{l.translate('templates.connect.title')}
						</h3>
						<p class="text-sm text-muted-foreground">
							{l.translate('templates.connect.description', [
								targets.find((t) => t.id === target)?.name ?? ''
							])}
						</p>
					</div>
					{#each [{ label: 'templates.connect.host', shown: host, value: host, copyLabel: 'templates.connect.copy_host' }, ...(connection ? [{ label: 'templates.connect.url', shown: shownConnection ?? '', value: connection, copyLabel: 'templates.connect.copy_url' }] : [])] as row (row.label)}
						<div class="flex items-center gap-2 rounded-md bg-sunken px-3 py-2.5">
							<span class="flex min-w-0 flex-1 flex-col">
								<span class="text-sm font-medium"
									>{l.translate(row.label as AppTranslationsString)}</span
								>
								<code class="font-mono text-xs/5 break-all text-muted-foreground">{row.shown}</code>
							</span>
							<IconButton
								type="button"
								icon={Copy01Icon}
								iconClass="size-4"
								class="size-8"
								label={l.translate(row.copyLabel as AppTranslationsString)}
								onclick={() => copy(row.value)}
							/>
						</div>
					{/each}
				</section>
			{/if}

			{#if signIn.length > 0}
				<section aria-labelledby="template-signin" class="flex flex-col gap-2">
					<div class="mb-1 flex flex-col">
						<h3 id="template-signin" class="text-sm font-semibold">
							{l.translate('templates.signin.title')}
						</h3>
						<p class="text-sm text-muted-foreground">
							{l.translate('templates.signin.description', [template.name])}
						</p>
					</div>
					<CredentialRows rows={signIn} />
				</section>
			{/if}

			<div class="flex flex-col">
				<span class="text-sm font-medium">{l.translate('templates.after')}</span>
				<p class="text-sm text-muted-foreground">{l.translate(template.after)}</p>
			</div>

			<p class="text-xs text-muted-foreground">
				{l.translate('templates.logo_credit', [template.logoCredit])} ·
				<a
					href={template.website}
					target="_blank"
					rel="noopener noreferrer"
					class="text-primary underline">{l.translate('templates.website')}</a
				>
				·
				<a
					href={template.docs}
					target="_blank"
					rel="noopener noreferrer"
					class="text-primary underline">{l.translate('templates.docs')}</a
				>
			</p>
		</fieldset>
	</form>

	{#snippet footer()}
		{#if reason}
			<span class="mr-auto text-sm text-muted-foreground max-md:basis-full">{reason}</span>
		{/if}
		<Button variant="outline" onclick={() => (open = false)}>{l.translate('cancel')}</Button>
		<Button
			type="submit"
			form={formId}
			loading={$loading}
			disabled={targets.length === 0}
			aria-disabled={!!reason || undefined}
			class={cn(reason && 'opacity-50')}
		>
			<Icon icon={Rocket01Icon} />
			{$loading ? l.translate('templates.deploying_label') : l.translate('templates.deploy')}
		</Button>
	{/snippet}
</Drawer>
