<script lang="ts">
	import { legalLinks, type LegalLink } from '../utils/legal';

	/** Imprint · Privacy · Terms, as far as the instance has them. */
	let { class: className = '' }: { class?: string } = $props();

	let links = $state<LegalLink[]>([]);
	legalLinks().then((loaded) => (links = loaded));
</script>

{#if links.length > 0}
	<nav aria-label="Legal" class="flex flex-wrap items-center gap-x-3 gap-y-1 {className}">
		{#each links as link (link.page)}
			<a
				href={link.href}
				class="hover:underline"
				target={link.external ? '_blank' : undefined}
				rel={link.external ? 'noopener noreferrer' : undefined}>{link.label}</a
			>
		{/each}
	</nav>
{/if}
