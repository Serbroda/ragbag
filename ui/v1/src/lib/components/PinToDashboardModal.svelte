<script lang="ts">
	import type { BookmarkDto, DashboardSectionDto } from 'ragbag-frontend-sdk';
	import { Button, Input, Label, Modal, Radio } from 'flowbite-svelte';
	import { errorMessage } from '../api/client';
	import { dashboardApi, pinToDashboard } from '../stores/dashboard';

	let {
		bookmark = $bindable(null),
		spaceId,
	}: { bookmark?: BookmarkDto | null; spaceId: string } = $props();

	const NEW = 'new';

	let open = $state(false);
	let sections = $state<{ index: number; section: DashboardSectionDto }[]>([]);
	let target = $state<string>(NEW);
	let newTitle = $state('Pinned');
	let error = $state('');
	let busy = $state(false);
	let done = $state(false);

	$effect(() => {
		if (bookmark) {
			open = true;
			load();
		}
	});

	$effect(() => {
		if (!open) bookmark = null;
	});

	async function load() {
		error = '';
		done = false;
		sections = [];
		try {
			const dashboard = await dashboardApi().getDashboard({ spaceId });
			sections = dashboard.sections
				.map((section, index) => ({ index, section }))
				.filter(({ section }) => section.type === 'LINKS');
			target = sections.length > 0 ? String(sections[0].index) : NEW;
		} catch (err) {
			error = await errorMessage(err, 'Failed to load your dashboard.');
		}
	}

	async function pin(e: Event) {
		e.preventDefault();
		if (!bookmark) return;
		busy = true;
		error = '';
		try {
			await pinToDashboard(spaceId, bookmark.id, target === NEW ? newTitle.trim() || 'Pinned' : Number(target));
			done = true;
			setTimeout(() => (open = false), 800);
		} catch (err) {
			error = await errorMessage(err, 'Failed to add the link to your dashboard.');
		} finally {
			busy = false;
		}
	}
</script>

<Modal bind:open size="sm" title="Add to dashboard">
	<form class="flex flex-col gap-3" onsubmit={pin}>
		<p class="truncate text-sm text-gray-500 dark:text-gray-400">{bookmark?.title || bookmark?.url}</p>
		{#each sections as { index, section } (index)}
			{@const already = section.bookmarks.some((b) => b.id === bookmark?.id)}
			<Radio name="pin-target" value={String(index)} bind:group={target} disabled={already}>
				{section.title}{already ? ' (already added)' : ''}
			</Radio>
		{/each}
		<Radio name="pin-target" value={NEW} bind:group={target}>New section</Radio>
		{#if target === NEW}
			<div class="ps-6">
				<Label for="pin-new-title" class="sr-only">Section title</Label>
				<Input id="pin-new-title" bind:value={newTitle} maxlength={80} />
			</div>
		{/if}
		{#if error}
			<p class="text-sm text-red-500 dark:text-red-400">{error}</p>
		{/if}
		<div class="flex justify-end gap-2">
			<Button color="alternative" onclick={() => (open = false)}>Cancel</Button>
			<Button type="submit" color={done ? 'green' : 'primary'} disabled={busy || done}>
				{done ? 'Added ✓' : busy ? 'Adding…' : 'Add'}
			</Button>
		</div>
	</form>
</Modal>
