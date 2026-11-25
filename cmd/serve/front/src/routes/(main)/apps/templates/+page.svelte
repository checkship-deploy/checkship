<!--
	Templates (DESIGN.md §11.1, §11.7 Templates gallery): New application › From a template. One list
	row per template (§5 Lists, the owner's pick "B · Rows" on 2026-11-25): the project's own logo in
	the tile, the name and whether it needs setup, then its category and one line. The row opens the
	review drawer, where Deploy creates the application and queues its first deployment. Search by
	name and filter by category.
-->
<script lang="ts">
	import Activity01Icon from '@hugeicons/core-free-icons/Activity01Icon';
	import ArrowRight01Icon from '@hugeicons/core-free-icons/ArrowRight01Icon';
	import CloudServerIcon from '@hugeicons/core-free-icons/CloudServerIcon';
	import CodeSquareIcon from '@hugeicons/core-free-icons/CodeSquareIcon';
	import Database01Icon from '@hugeicons/core-free-icons/Database01Icon';
	import FileSearchIcon from '@hugeicons/core-free-icons/FileSearchIcon';
	import LayoutGridIcon from '@hugeicons/core-free-icons/LayoutGridIcon';
	import Mail01Icon from '@hugeicons/core-free-icons/Mail01Icon';
	import MessageMultiple01Icon from '@hugeicons/core-free-icons/MessageMultiple01Icon';
	import Note01Icon from '@hugeicons/core-free-icons/Note01Icon';
	import Notification01Icon from '@hugeicons/core-free-icons/Notification01Icon';
	import Search01Icon from '@hugeicons/core-free-icons/Search01Icon';
	import Shield01Icon from '@hugeicons/core-free-icons/Shield01Icon';
	import UserLock01Icon from '@hugeicons/core-free-icons/UserLock01Icon';
	import EmptyState from '#lib/components/empty-state.svelte';
	import Icon from '#lib/components/icon.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import FilterMenu, { type FilterOption } from '#lib/components/table/filter-menu.svelte';
	import TimelineList from '#lib/components/timeline/timeline-list.svelte';
	import TimelineRow from '#lib/components/timeline/timeline-row.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import { categories, templates } from '#lib/templates/catalog.ts';
	import type { Template, TemplateCategory } from '#lib/templates/types.ts';
	import { needsSetup } from '#lib/templates/values.ts';
	import TemplateLogo from '#lib/templates/template-logo.svelte';
	import TemplateDrawer from './template-drawer.svelte';

	let { data } = $props();

	let search = $state('');
	let category = $state<'all' | TemplateCategory>('all');

	const categoryIcons: Record<TemplateCategory, typeof LayoutGridIcon> = {
		monitoring: Activity01Icon,
		notes: Note01Icon,
		developer: CodeSquareIcon,
		security: Shield01Icon,
		notifications: Notification01Icon,
		database: Database01Icon,
		email: Mail01Icon,
		search: FileSearchIcon,
		storage: CloudServerIcon,
		messaging: MessageMultiple01Icon,
		auth: UserLock01Icon
	};

	const categoryLabel = (c: TemplateCategory) => l.translate(`templates.category.${c}`);

	const categoryOptions: FilterOption<'all' | TemplateCategory>[] = [
		{ value: 'all', label: l.translate('templates.category.all'), icon: LayoutGridIcon },
		...categories.map((c) => ({ value: c, label: categoryLabel(c), icon: categoryIcons[c] }))
	];

	const shown = $derived(
		templates.filter((t) => {
			const q = search.trim().toLowerCase();
			const text = `${t.name} ${categoryLabel(t.category)} ${l.translate(t.description)}`;
			return (category === 'all' || t.category === category) && text.toLowerCase().includes(q);
		})
	);

	let selected = $state<Template>();
	let drawerOpen = $state(false);

	function open(template: Template) {
		selected = template;
		drawerOpen = true;
	}

	function label(t: Template): string {
		return [
			t.name,
			needsSetup(t) ? l.translate('templates.needs_setup') : l.translate('templates.ready'),
			categoryLabel(t.category),
			l.translate(t.description)
		].join(', ');
	}
</script>

<PageHeader
	title={l.translate('templates.title')}
	crumbs={[{ href: routes.apps, label: l.translate('breadcrumb.applications') }]}
>
	{#snippet status()}
		<span>{l.translate('templates.summary', [templates.length])}</span>
	{/snippet}
</PageHeader>

<div class="flex flex-col gap-3">
	<div class="flex flex-wrap items-center gap-2">
		<div class="relative w-72 max-w-full">
			<Input
				type="search"
				bind:value={search}
				placeholder={l.translate('templates.search')}
				aria-label={l.translate('templates.search')}
				class="pr-9"
			/>
			<Icon
				icon={Search01Icon}
				class="pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2 text-muted-foreground"
			/>
		</div>
		<FilterMenu
			label={l.translate('templates.category')}
			icon={LayoutGridIcon}
			bind:value={category}
			options={categoryOptions}
		/>
	</div>

	{#if shown.length > 0}
		<TimelineList label={l.translate('templates.title')}>
			{#each shown as template (template.id)}
				<TimelineRow
					label={label(template)}
					onclick={() => open(template)}
					selected={drawerOpen && selected?.id === template.id}
				>
					{#snippet tile()}
						<TemplateLogo {template} />
					{/snippet}
					{#snippet title()}{template.name}{/snippet}
					{#snippet badge()}
						{#if needsSetup(template)}
							<StatusBadge state="setup" label={l.translate('templates.needs_setup')} />
						{:else}
							<StatusBadge state="ready" label={l.translate('templates.ready')} />
						{/if}
					{/snippet}
					{#snippet details()}
						{categoryLabel(template.category)} · {l.translate(template.description)}
					{/snippet}
					{#snippet actions()}
						<span
							aria-hidden="true"
							class="pointer-events-none flex items-center gap-2 text-muted-foreground"
						>
							<span class="hidden font-mono text-xs md:inline">{template.image}</span>
							<Icon icon={ArrowRight01Icon} class="size-4" />
						</span>
					{/snippet}
				</TimelineRow>
			{/each}
		</TimelineList>
	{:else}
		<EmptyState
			illustration="search"
			size="section"
			title={l.translate('templates.no_match.title')}
			description={l.translate('templates.no_match.description')}
		>
			{#snippet action()}
				<Button
					variant="outline"
					onclick={() => {
						search = '';
						category = 'all';
					}}
				>
					{l.translate('list.clear_filters')}
				</Button>
			{/snippet}
		</EmptyState>
	{/if}

	<p class="mt-4 text-xs text-muted-foreground">{l.translate('templates.logos_notice')}</p>
</div>

{#if selected}
	{#key selected.id}
		<TemplateDrawer
			bind:open={drawerOpen}
			template={selected}
			targets={data.targets}
			apps={data.apps}
		/>
	{/key}
{/if}
