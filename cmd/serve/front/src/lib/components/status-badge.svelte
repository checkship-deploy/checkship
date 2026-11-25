<!--
	State badge (DESIGN.md §4, §11.4): soft fill, ink text, the colour only on the icon, 4px corner.
	Meaning never rests on the colour: the icon and the words carry it.
-->
<script lang="ts" module>
	export type BadgeState =
		| 'succeeded'
		| 'ready'
		| 'failed'
		| 'running'
		| 'configuring'
		| 'pending'
		| 'deleting'
		| 'cancelled'
		/** A template that asks for settings before it can deploy. */
		| 'setup';
</script>

<script lang="ts">
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import CancelCircleIcon from '@hugeicons/core-free-icons/CancelCircleIcon';
	import CheckmarkCircle02Icon from '@hugeicons/core-free-icons/CheckmarkCircle02Icon';
	import Clock01Icon from '@hugeicons/core-free-icons/Clock01Icon';
	import Delete02Icon from '@hugeicons/core-free-icons/Delete02Icon';
	import Loading03Icon from '@hugeicons/core-free-icons/Loading03Icon';
	import SlidersHorizontalIcon from '@hugeicons/core-free-icons/SlidersHorizontalIcon';
	import UnavailableIcon from '@hugeicons/core-free-icons/UnavailableIcon';
	import type { IconData } from '#lib/components/icon.svelte';
	import Icon from '#lib/components/icon.svelte';
	import { cn } from '#lib/utils.js';

	type Props = { state: BadgeState; label: string; class?: string };

	let { state, label, class: className }: Props = $props();

	const styles: Record<BadgeState, { fill: string; icon: IconData; tone: string; spin?: boolean }> =
		{
			succeeded: { fill: 'bg-success-soft', icon: CheckmarkCircle02Icon, tone: 'text-success' },
			ready: { fill: 'bg-success-soft', icon: CheckmarkCircle02Icon, tone: 'text-success' },
			failed: { fill: 'bg-danger-soft', icon: CancelCircleIcon, tone: 'text-danger' },
			running: { fill: 'bg-warning-soft', icon: Loading03Icon, tone: 'text-warning', spin: true },
			configuring: {
				fill: 'bg-warning-soft',
				icon: Loading03Icon,
				tone: 'text-warning',
				spin: true
			},
			pending: { fill: 'bg-raised', icon: Clock01Icon, tone: 'text-muted-foreground' },
			cancelled: { fill: 'bg-raised', icon: UnavailableIcon, tone: 'text-muted-foreground' },
			deleting: { fill: 'bg-danger-soft', icon: Delete02Icon, tone: 'text-danger' },
			setup: { fill: 'bg-raised', icon: SlidersHorizontalIcon, tone: 'text-muted-foreground' }
		};

	const style = $derived(
		styles[state] ?? { fill: 'bg-raised', icon: Alert02Icon, tone: 'text-ink' }
	);
</script>

<span
	class={cn(
		'inline-flex h-6 items-center gap-1.5 rounded-sm px-2 text-xs font-semibold whitespace-nowrap text-ink',
		style.fill,
		className
	)}
>
	<Icon icon={style.icon} class={cn('size-3.5', style.tone, style.spin && 'animate-spin')} />
	{label}
</span>
