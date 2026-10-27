<script lang="ts">
	import type { SpaceMemberRoleEnum } from 'ragbag-frontend-sdk';

	let { role }: { role: SpaceMemberRoleEnum } = $props();

	const STYLES: Record<SpaceMemberRoleEnum, string> = {
		OWNER: 'bg-primary-500/15 text-primary-700 ring-primary-500/30 dark:text-primary-300',
		ADMIN: 'bg-sky-500/15 text-sky-700 ring-sky-500/30 dark:text-sky-300',
		CONTRIBUTOR: 'bg-emerald-500/15 text-emerald-700 ring-emerald-500/30 dark:text-emerald-300',
		VIEWER: 'bg-gray-500/15 text-gray-700 ring-gray-500/30 dark:text-gray-300',
	};
</script>

<span class="inline-flex shrink-0 items-center rounded-md px-2 py-0.5 text-xs font-medium capitalize ring-1 ring-inset {STYLES[role]}">
	{role.toLowerCase()}
</span>
