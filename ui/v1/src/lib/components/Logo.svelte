<script lang="ts">
	let { size = 32, class: className = '' }: { size?: number; class?: string } = $props();
	const id = `logo-${Math.random().toString(36).slice(2, 8)}`;
</script>

<svg width={size} height={size} viewBox="0 0 64 64" class={className} aria-hidden="true">
	<defs>
		<linearGradient {id} x1="0" y1="0" x2="1" y2="1">
			<stop offset="0" stop-color="#fbbf24" />
			<stop offset="1" stop-color="#d97706" />
		</linearGradient>
	</defs>
	<rect width="64" height="64" rx="16" fill="url(#{id})" />
	<!-- a backpack whose bottom is a bookmark -->
	<path d="M27 16v-3a5 5 0 0 1 10 0v3" fill="none" stroke="#1c1917" stroke-width="3.5" />
	<path d="M17 28a13 13 0 0 1 13-13h4a13 13 0 0 1 13 13v24l-15-7-15 7z" fill="#1c1917" />
	<path d="M24 17c-4 10-4 20-1 29M40 17c4 10 4 20 1 29" fill="none" stroke="url(#{id})" stroke-width="3" stroke-linecap="round" />
	<path d="M19.5 40h5.5M39 40h5.5" stroke="#1c1917" stroke-width="1.6" />
</svg>
