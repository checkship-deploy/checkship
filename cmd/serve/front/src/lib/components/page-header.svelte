<!--
	Page header (DESIGN.md §5): as tall as its buttons (36px), so the workspace's 40px padding is the gap above them; the title is the page's one h1 (20/28), optionally after a
	breadcrumb; actions sit on the right and wrap, one primary. A status line follows, then 30px
	before the body. No line under it.
-->
<script lang="ts">
	import ArrowRight01Icon from '@hugeicons/core-free-icons/ArrowRight01Icon';
	import type { Snippet } from 'svelte';
	import Icon from '#lib/components/icon.svelte';

	type Crumb = { href: string; label: string };
	type Props = {
		title: string;
		crumbs?: Crumb[];
		actions?: Snippet;
		status?: Snippet;
		/** Drawn before the title, such as the logo of the template an application runs. */
		leading?: Snippet;
	};

	let { title, crumbs = [], actions, status, leading }: Props = $props();
</script>

<svelte:head>
	<title>checkship - {title}</title>
</svelte:head>

<header class="flex min-h-9 flex-wrap items-center justify-between gap-x-3 gap-y-2">
	{#if crumbs.length > 0}
		<nav aria-label="Breadcrumb" class="flex min-w-0 flex-wrap items-center gap-x-2 text-sm">
			{#each crumbs as crumb (crumb.href)}
				<a
					href={crumb.href}
					class="rounded-sm text-muted-foreground outline-none hover:text-ink focus-visible:ring-2 focus-visible:ring-ring"
				>
					{crumb.label}
				</a>
				<Icon icon={ArrowRight01Icon} class="size-3.5 text-muted-foreground" />
			{/each}
			<span class="flex min-w-0 items-center gap-2">
				{@render leading?.()}
				<h1 aria-current="page" class="min-w-0 truncate text-xl/7 font-semibold tracking-tight">
					{title}
				</h1>
			</span>
		</nav>
	{:else}
		<span class="flex min-w-0 items-center gap-2">
			{@render leading?.()}
			<h1 class="min-w-0 truncate text-xl/7 font-semibold tracking-tight">{title}</h1>
		</span>
	{/if}
	{#if actions}
		<div class="flex flex-wrap items-center gap-1.5">{@render actions()}</div>
	{/if}
</header>

{#if status}
	<div class="flex flex-wrap items-center gap-x-3 gap-y-1.5 pb-7.5 text-sm text-muted-foreground">
		{@render status()}
	</div>
{:else}
	<div class="pb-6"></div>
{/if}
