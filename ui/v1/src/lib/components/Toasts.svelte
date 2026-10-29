<script lang="ts">
	import { CloseOutline } from 'flowbite-svelte-icons';
	import { dismissToast, toasts } from '../stores/toast.svelte';
</script>

<div class="pointer-events-none fixed inset-x-0 bottom-4 z-[60] flex flex-col items-center gap-2 px-4" aria-live="polite">
	{#each toasts.list as toast (toast.id)}
		<div
			class="pointer-events-auto flex max-w-full items-center gap-3 rounded-lg bg-gray-950 py-2 pl-4 pr-2 text-sm text-gray-100 shadow-2xl ring-1 ring-white/10"
			role="status"
		>
			<span class="min-w-0 truncate">{toast.text}</span>
			{#if toast.action}
				<button
					type="button"
					class="shrink-0 rounded px-2 py-1 font-semibold text-primary-400 hover:bg-white/10"
					onclick={() => {
						toast.action!.run();
						dismissToast(toast.id);
					}}
				>
					{toast.action.label}
				</button>
			{/if}
			<button
				type="button"
				class="shrink-0 rounded p-1 text-gray-400 hover:bg-white/10 hover:text-gray-200"
				aria-label="Dismiss"
				onclick={() => dismissToast(toast.id)}
			>
				<CloseOutline class="h-4 w-4" />
			</button>
		</div>
	{/each}
</div>
