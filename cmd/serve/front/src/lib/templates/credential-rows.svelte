<!--
	Sign-in and connection values, one `sunken` well each (DESIGN.md §11.7): the label, the value in
	mono, a Show/Hide button for a secret (masked until asked) and Copy, which always copies the whole
	value. A row with `href` is a link, such as Vaultwarden's admin page. Used by the template drawer
	before deploy and by the application's Sign-in and connection drawer after.
-->
<script lang="ts" module>
	export type CredentialRow = {
		label: string;
		value: string;
		/** Masked until revealed: the whole value, or only `secrets` inside it. */
		secret?: boolean;
		/** The secrets inside a longer value, such as a connection string's password. */
		secrets?: string[];
		/** Opens the value, such as an admin page. */
		href?: string;
		/** Shown under the value. */
		help?: string;
	};
</script>

<script lang="ts">
	import Copy01Icon from '@hugeicons/core-free-icons/Copy01Icon';
	import LinkSquare02Icon from '@hugeicons/core-free-icons/LinkSquare02Icon';
	import ViewIcon from '@hugeicons/core-free-icons/ViewIcon';
	import ViewOffIcon from '@hugeicons/core-free-icons/ViewOffIcon';
	import { toast } from 'svelte-sonner';
	import { copyText } from '#lib/clipboard.ts';
	import Icon from '#lib/components/icon.svelte';
	import IconButton from '#lib/components/icon-button.svelte';
	import l from '#lib/localization/index.ts';

	type Props = { rows: CredentialRow[] };

	let { rows }: Props = $props();

	let revealed = $state<Record<string, boolean>>({});

	function masked(value: string): string {
		return value.length > 12 ? `${value.slice(0, 4)}••••••••${value.slice(-4)}` : '••••••••';
	}

	/** The value as shown while hidden: its secrets masked, or all of it. */
	function hide(row: CredentialRow): string {
		const inside = (row.secrets ?? []).filter((secret) => !!secret && row.value.includes(secret));
		if (inside.length === 0) return masked(row.value);
		return inside.reduce((shown, secret) => shown.replaceAll(secret, masked(secret)), row.value);
	}

	async function copy(value: string) {
		if (await copyText(value)) toast.success(l.translate('copied'));
		else toast.error(l.translate('copy_blocked'));
	}
</script>

<div class="flex flex-col gap-2">
	{#each rows as row (row.label)}
		{@const shown = row.secret && !revealed[row.label] ? hide(row) : row.value}
		<div class="flex items-center gap-2 rounded-md bg-sunken px-3 py-2.5">
			<span class="flex min-w-0 flex-1 flex-col">
				<span class="text-sm font-medium">{row.label}</span>
				{#if row.href}
					<a
						href={row.href}
						target="_blank"
						rel="noopener noreferrer"
						class="flex min-w-0 items-center gap-1 font-mono text-xs/5 break-all text-primary hover:underline"
					>
						{shown}
						<Icon icon={LinkSquare02Icon} class="size-3 shrink-0" />
					</a>
				{:else}
					<code class="font-mono text-xs/5 break-all text-muted-foreground">{shown}</code>
				{/if}
				{#if row.help}<span class="text-xs text-muted-foreground">{row.help}</span>{/if}
			</span>
			{#if row.secret}
				<IconButton
					type="button"
					icon={revealed[row.label] ? ViewOffIcon : ViewIcon}
					iconClass="size-4"
					class="size-8"
					label={revealed[row.label]
						? l.translate('templates.hide', [row.label])
						: l.translate('templates.reveal', [row.label])}
					aria-pressed={!!revealed[row.label]}
					onclick={() => (revealed[row.label] = !revealed[row.label])}
				/>
			{/if}
			<IconButton
				type="button"
				icon={Copy01Icon}
				iconClass="size-4"
				class="size-8"
				label={l.translate('templates.copy', [row.label])}
				onclick={() => copy(row.value)}
			/>
		</div>
	{/each}
</div>
