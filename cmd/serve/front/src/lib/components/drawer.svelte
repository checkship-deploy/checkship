<!--
	Right drawer (DESIGN.md §11.6): for short tasks where the page behind still matters. 560px for
	forms, 620px for a form with tables and steps (Domains), 680px for previews, full screen on phones. Header (title, one line, close), a body that
	scrolls, and a footer with the actions on the right. A shadow appears above the footer only while
	there is more to scroll. Esc and the scrim close it; focus returns to what opened it.
-->
<script lang="ts">
	import Cancel01Icon from '@hugeicons/core-free-icons/Cancel01Icon';
	import type { Snippet } from 'svelte';
	import Icon from '#lib/components/icon.svelte';
	import { iconButtonClass } from '#lib/components/icon-button.svelte';
	import * as Sheet from '#lib/components/ui/sheet/index.js';
	import * as Tooltip from '#lib/components/ui/tooltip/index.js';
	import l from '#lib/localization/index.ts';
	import { cn } from '#lib/utils.js';

	type Props = {
		open: boolean;
		title: string;
		description?: string | Snippet;
		size?: 'form' | 'wide' | 'preview';
		/** Before the title: a picture that names what the drawer is about (a template's logo). */
		leading?: Snippet;
		children: Snippet;
		footer?: Snippet;
		onclose?: () => void;
	};

	let {
		open = $bindable(false),
		title,
		description,
		size = 'form',
		leading,
		children,
		footer,
		onclose
	}: Props = $props();

	let body = $state<HTMLDivElement>();
	let moreBelow = $state(false);

	function measure() {
		if (!body) return;
		moreBelow = body.scrollTop + body.clientHeight < body.scrollHeight - 1;
	}

	// Content changes size (a source switch, a file dropped): measure again.
	$effect(() => {
		if (!body) return;
		const observer = new ResizeObserver(measure);
		observer.observe(body);
		for (const child of body.children) observer.observe(child);
		return () => observer.disconnect();
	});
</script>

<Sheet.Root
	bind:open
	onOpenChange={(value) => {
		if (!value) onclose?.();
	}}
>
	<Sheet.Content
		side="right"
		showCloseButton={false}
		class={cn('md:w-140', size === 'wide' && 'md:w-155', size === 'preview' && 'md:w-170')}
	>
		<header class="flex shrink-0 items-start justify-between gap-4 pt-5 pr-4 pb-4 pl-6">
			<div class="flex min-w-0 items-center gap-3.5">
				{@render leading?.()}
				<div class="flex min-w-0 flex-col gap-1">
					<Sheet.Title class="text-lg/[26px] font-semibold tracking-tight">{title}</Sheet.Title>
					{#if description}
						<Sheet.Description class="text-sm text-muted-foreground">
							{#if typeof description === 'string'}{description}{:else}{@render description()}{/if}
						</Sheet.Description>
					{/if}
				</div>
			</div>
			<Tooltip.Root>
				<Tooltip.Trigger>
					{#snippet child({ props: tooltipProps })}
						<Sheet.Close {...tooltipProps} class={iconButtonClass({ touch: true })}>
							<Icon icon={Cancel01Icon} class="size-4.5" />
							<span class="sr-only">{l.translate('close')}</span>
						</Sheet.Close>
					{/snippet}
				</Tooltip.Trigger>
				<Tooltip.Content side="bottom">{l.translate('close')}</Tooltip.Content>
			</Tooltip.Root>
		</header>

		<div
			bind:this={body}
			onscroll={measure}
			class="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto px-4 pt-1 pb-6 md:px-6"
		>
			{@render children()}
		</div>

		{#if footer}
			<footer
				class={cn(
					'flex shrink-0 flex-wrap items-center justify-end gap-2 px-4 pt-3 pb-7 transition-shadow md:px-6 md:py-4 [&>*]:max-md:h-11 [&>*]:max-md:flex-1',
					moreBelow && 'shadow-[0_-16px_24px_-16px_rgb(0_0_0/0.7)]'
				)}
			>
				{@render footer()}
			</footer>
		{/if}
	</Sheet.Content>
</Sheet.Root>
