<script lang="ts">
	import type { SpaceMemberDto } from 'ragbag-frontend-sdk';
	import { Checkbox } from 'flowbite-svelte';
	import Avatar from './Avatar.svelte';
	import RoleBadge from './RoleBadge.svelte';

	let {
		members,
		selected = $bindable(new Set<string>()),
		currentUserId,
		disabled = false,
	}: {
		members: SpaceMemberDto[];
		selected?: Set<string>;
		/** Always included and cannot be unchecked: nobody can lock themselves out. */
		currentUserId?: string | null;
		disabled?: boolean;
	} = $props();

	function toggle(userId: string) {
		const next = new Set(selected);
		if (next.has(userId)) next.delete(userId);
		else next.add(userId);
		selected = next;
	}
</script>

<ul class="-mx-2 max-h-64 overflow-y-auto">
	{#each members as member (member.userId)}
		{@const self = member.userId === currentUserId}
		<li>
			<label
				class="flex items-center gap-3 rounded-lg px-2 py-1.5 {disabled || self
					? 'cursor-default'
					: 'cursor-pointer hover:bg-gray-100 dark:hover:bg-gray-700/60'}"
			>
				<Checkbox
					checked={self || selected.has(member.userId)}
					disabled={disabled || self}
					onchange={() => toggle(member.userId)}
				/>
				<Avatar name={member.username} imageId={member.avatarId} class="h-7 w-7 rounded-full text-[10px]" />
				<span class="min-w-0 flex-1 truncate text-sm text-gray-900 dark:text-white">
					{member.username}
					{#if self}<span class="text-gray-500 dark:text-gray-400">(you)</span>{/if}
				</span>
				<RoleBadge role={member.role} />
			</label>
		</li>
	{/each}
</ul>
