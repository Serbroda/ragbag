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
	<path d="M22 14h20a3 3 0 0 1 3 3v33l-13-8-13 8V17a3 3 0 0 1 3-3z" fill="#1c1917" />
</svg>
