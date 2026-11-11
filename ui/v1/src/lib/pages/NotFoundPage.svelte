<script lang="ts">
	import { Button } from 'flowbite-svelte';
	import CenteredScreen from '../components/CenteredScreen.svelte';
	import { p } from '../router';

	/** An address no page belongs to (signed-out visitors never get here: they are sent to the login). */
</script>

<svelte:head>
	<title>Page not found · Ragbag</title>
</svelte:head>

<CenteredScreen subtitle="This page does not exist.">
	<p class="mb-6 text-center text-sm text-gray-500 dark:text-gray-400">
		The address may be mistyped, or what it led to was moved or deleted.
	</p>
	<div class="flex justify-center">
		<Button href={p('/')}>Back to Ragbag</Button>
	</div>
</CenteredScreen>
