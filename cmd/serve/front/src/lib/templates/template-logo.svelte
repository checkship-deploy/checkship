<!--
	A template's logo as the project publishes it (static/template-logos/, NOTICE.md): its dark
	variant in dark mode when it has one, or a white plate behind a dark mark that has none. Shown
	with <img>, so nothing in an SVG runs; decorative, since the name always sits beside it.
-->
<script lang="ts">
	import { cn } from '#lib/utils.js';
	import type { Template } from './types.ts';

	type Props = {
		template: Pick<Template, 'logo' | 'logoDark' | 'logoPlate'>;
		/** The box the logo fits in, such as size-8. */
		class?: string;
	};

	let { template, class: className = 'size-8' }: Props = $props();
</script>

{#if template.logoPlate}
	<span class={cn('flex shrink-0 items-center justify-center rounded-md dark:bg-white', className)}>
		<img src={template.logo} alt="" class="size-full object-contain dark:size-3/4" />
	</span>
{:else}
	<img
		src={template.logo}
		alt=""
		class={cn('shrink-0 object-contain', className, template.logoDark && 'dark:hidden')}
	/>
	{#if template.logoDark}
		<img
			src={template.logoDark}
			alt=""
			class={cn('hidden shrink-0 object-contain dark:block', className)}
		/>
	{/if}
{/if}
