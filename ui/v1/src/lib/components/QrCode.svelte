<script lang="ts">
	/** A QR code drawn as SVG in the browser; the library is only loaded when one is shown. */
	let { value, label }: { value: string; label: string } = $props();

	let size = $state(0);
	let path = $state('');

	$effect(() => {
		const text = value;
		import('qrcode-generator').then(({ default: qrcode }) => {
			if (text !== value) return;
			const code = qrcode(0, 'M');
			code.addData(text);
			code.make();
			const count = code.getModuleCount();
			let d = '';
			for (let row = 0; row < count; row++) {
				for (let col = 0; col < count; col++) {
					if (code.isDark(row, col)) d += `M${col} ${row}h1v1h-1z`;
				}
			}
			size = count;
			path = d;
		});
	});
</script>

<!-- Always dark on white with a quiet zone, whatever the theme: scanners need the contrast -->
<div class="inline-block rounded-lg bg-white p-3">
	{#if size}
		<svg viewBox="0 0 {size} {size}" class="h-44 w-44" role="img" aria-label={label} shape-rendering="crispEdges">
			<path d={path} fill="#000" />
		</svg>
	{:else}
		<div class="h-44 w-44"></div>
	{/if}
</div>
