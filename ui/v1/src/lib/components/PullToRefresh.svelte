<script lang="ts">
	/**
	 * Pull-to-refresh for touch devices: the installed app on iOS has no reload button, and the browser's own
	 * gesture never starts because the page itself does not scroll (only `root` does). Pulling `root` down from the
	 * very top reloads the page, which also picks up a new frontend after an update.
	 */
	import { ArrowDownOutline } from 'flowbite-svelte-icons';
	import { Spinner } from 'flowbite-svelte';

	let { root }: { root: HTMLElement | undefined } = $props();

	/** Finger travel needed to reload; the indicator moves half of it. */
	const THRESHOLD = 140;
	const MAX_OFFSET = 100;

	let pull = $state(0);
	let reloading = $state(false);
	let offset = $derived(Math.min(pull / 2, MAX_OFFSET));
	let armed = $derived(pull >= THRESHOLD);

	/** Whether the touch began inside something that scrolls on its own or a dialog, which must keep the gesture. */
	function ownedElsewhere(target: EventTarget | null, root: HTMLElement): boolean {
		for (let el = target as HTMLElement | null; el && el !== root; el = el.parentElement) {
			if (el.matches('dialog, [role="dialog"], input, textarea, select, [contenteditable]')) return true;
			if (el.scrollTop > 0 && el.scrollHeight > el.clientHeight) return true;
		}
		return false;
	}

	$effect(() => {
		const el = root;
		if (!el || !matchMedia('(pointer: coarse)').matches) return;

		let startX = 0;
		let startY = 0;
		/** null: undecided, true: pulling, false: an ordinary scroll or swipe, ignore until the finger lifts */
		let tracking: boolean | null = false;

		const onStart = (e: TouchEvent) => {
			tracking = false;
			if (reloading || e.touches.length !== 1 || el.scrollTop > 0 || ownedElsewhere(e.target, el)) return;
			startX = e.touches[0].clientX;
			startY = e.touches[0].clientY;
			tracking = null;
		};
		const onMove = (e: TouchEvent) => {
			if (tracking === false) return;
			if (e.touches.length !== 1) return reset();
			const dx = e.touches[0].clientX - startX;
			const dy = e.touches[0].clientY - startY;
			if (tracking === null) {
				if (Math.abs(dx) < 8 && Math.abs(dy) < 8) return;
				// Only a downward, mostly vertical drag from the top is ours
				tracking = dy > 0 && Math.abs(dy) > Math.abs(dx) && el.scrollTop <= 0;
				if (!tracking) return;
			}
			if (e.cancelable) e.preventDefault();
			pull = Math.max(0, dy);
		};
		const onEnd = () => {
			if (tracking && armed) {
				reloading = true;
				pull = THRESHOLD;
				location.reload();
				return;
			}
			reset();
		};
		const reset = () => {
			tracking = false;
			pull = 0;
		};

		el.addEventListener('touchstart', onStart, { passive: true });
		// Not passive: while pulling, the default (iOS rubber band) has to be suppressed
		el.addEventListener('touchmove', onMove, { passive: false });
		el.addEventListener('touchend', onEnd);
		el.addEventListener('touchcancel', reset);
		return () => {
			el.removeEventListener('touchstart', onStart);
			el.removeEventListener('touchmove', onMove);
			el.removeEventListener('touchend', onEnd);
			el.removeEventListener('touchcancel', reset);
		};
	});
</script>

{#if pull > 0 || reloading}
	<div
		class="pointer-events-none absolute inset-x-0 top-0 z-30 flex justify-center"
		style:transform="translateY({offset - 40}px)"
		style:transition={pull > 0 && !reloading ? 'none' : undefined}
		aria-hidden="true"
	>
		<div
			class="flex h-9 w-9 items-center justify-center rounded-full bg-white text-gray-600 shadow-md ring-1 ring-gray-200 dark:bg-gray-800 dark:text-gray-300 dark:ring-gray-700"
			style:opacity={Math.min(1, offset / 40)}
		>
			{#if reloading}
				<Spinner size="4" />
			{:else}
				<ArrowDownOutline
					class="h-4 w-4 transition-transform duration-150 motion-reduce:transition-none {armed
						? 'rotate-180 text-primary-500'
						: ''}"
				/>
			{/if}
		</div>
	</div>
{/if}
