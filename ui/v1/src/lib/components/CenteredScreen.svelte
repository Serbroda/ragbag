<script lang="ts">
	import type { Snippet } from 'svelte';
	import Logo from './Logo.svelte';

	let { children, subtitle }: { children: Snippet; subtitle?: string } = $props();
</script>

<!-- Shared frame for the screens outside the app shell (login, invites, welcome) -->
<div class="relative flex min-h-dvh items-center justify-center overflow-hidden bg-gray-950 p-4 text-gray-100">
	<div
		class="pointer-events-none absolute left-1/2 top-[-20%] h-[600px] w-[900px] -translate-x-1/2 rounded-full bg-primary-500/15 blur-3xl"
		aria-hidden="true"
	></div>
	<div class="relative w-full max-w-md">
		<div class="mb-6 flex flex-col items-center gap-3 text-center">
			<Logo size={56} />
			<h1 class="text-2xl font-bold tracking-tight text-white">Ragbag</h1>
			{#if subtitle}
				<p class="text-sm text-gray-400">{subtitle}</p>
			{/if}
		</div>
		<div class="rounded-2xl border border-white/10 bg-gray-900/80 p-8 shadow-2xl backdrop-blur">
			{@render children()}
		</div>
	</div>
</div>
