<script lang="ts">
	import { SpaceApi } from 'ragbag-frontend-sdk';
	import type { SpaceMemberDto, SpaceMemberRoleEnum } from 'ragbag-frontend-sdk';
	import { apiConfig } from '../api/client';
	import { colorFor, initials } from '../utils/format';

	let { spaceId, currentUserId }: { spaceId: string; currentUserId?: string } = $props();

	const GROUPS: { role: SpaceMemberRoleEnum; label: string }[] = [
		{ role: 'OWNER', label: 'Owners' },
		{ role: 'ADMIN', label: 'Admins' },
		{ role: 'CONTRIBUTOR', label: 'Contributors' },
		{ role: 'VIEWER', label: 'Viewers' },
	];

	let members = $state<SpaceMemberDto[]>([]);
	let failed = $state(false);

	$effect(() => {
		const id = spaceId;
		failed = false;
		new SpaceApi(apiConfig())
			.getSpaceMembers({ spaceId: id })
			.then((result) => {
				if (id === spaceId) members = result;
			})
			.catch(() => (failed = true));
	});
</script>

<div class="h-full overflow-y-auto px-3 py-4">
	{#if failed}
		<p class="px-2 text-sm text-gray-500 dark:text-gray-400">Members could not be loaded.</p>
	{/if}
	{#each GROUPS as group (group.role)}
		{@const inGroup = members.filter((m) => m.role === group.role)}
		{#if inGroup.length > 0}
			<h3 class="mb-1 mt-3 px-2 text-xs font-semibold uppercase tracking-wider text-gray-500 first:mt-0 dark:text-gray-400">
				{group.label} — {inGroup.length}
			</h3>
			<ul>
				{#each inGroup as member (member.userId)}
					<li class="flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-gray-100 dark:hover:bg-gray-700">
						<span
							class="flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-xs font-semibold text-white {colorFor(
								member.username,
							)}"
						>
							{initials(member.username)}
						</span>
						<span class="truncate text-sm text-gray-700 dark:text-gray-200">
							{member.username}
							{#if member.userId === currentUserId}
								<span class="text-xs text-gray-400">(you)</span>
							{/if}
						</span>
					</li>
				{/each}
			</ul>
		{/if}
	{/each}
</div>
