<!--
	New application (DESIGN.md §11.7, New application menu): the page's primary button opens a menu
	with the two ways to start, built like the split Deploy menu (two-line items with an icon).
	From scratch is the long form at /apps/new; From a template is the gallery at /apps/templates.
-->
<script lang="ts">
	import Add01Icon from '@hugeicons/core-free-icons/Add01Icon';
	import ArrowDown01Icon from '@hugeicons/core-free-icons/ArrowDown01Icon';
	import DashboardSquareAddIcon from '@hugeicons/core-free-icons/DashboardSquareAddIcon';
	import File01Icon from '@hugeicons/core-free-icons/File01Icon';
	import { goto } from '$app/navigation';
	import Icon from '#lib/components/icon.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';

	type Props = { align?: 'start' | 'center' | 'end' };

	let { align = 'end' }: Props = $props();

	const ways = [
		{
			href: routes.createApp,
			icon: File01Icon,
			title: l.translate('app.new.scratch'),
			hint: l.translate('app.new.scratch.hint')
		},
		{
			href: routes.templates,
			icon: DashboardSquareAddIcon,
			title: l.translate('app.new.template'),
			hint: l.translate('app.new.template.hint')
		}
	];
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger>
		{#snippet child({ props })}
			<Button {...props}>
				<Icon icon={Add01Icon} />
				{l.translate('app.new')}
				<Icon icon={ArrowDown01Icon} class="size-4" />
			</Button>
		{/snippet}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content {align} sideOffset={6} class="w-74 p-1.5">
		<DropdownMenu.Label class="px-2.5 pt-2 pb-1 text-xs font-semibold text-muted-foreground">
			{l.translate('app.new.menu')}
		</DropdownMenu.Label>
		{#each ways as way (way.href)}
			<DropdownMenu.Item class="min-h-11 gap-2.5 px-2.5 py-1.5" onSelect={() => goto(way.href)}>
				<Icon icon={way.icon} class="text-muted-foreground" />
				<span class="flex flex-col">
					<span class="text-label font-medium">{way.title}</span>
					<span class="text-xs text-muted-foreground">{way.hint}</span>
				</span>
			</DropdownMenu.Item>
		{/each}
	</DropdownMenu.Content>
</DropdownMenu.Root>
