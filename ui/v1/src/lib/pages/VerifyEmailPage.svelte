<script lang="ts">
	import CenteredScreen from '../components/CenteredScreen.svelte';
	import { errorMessage, verifyEmail } from '../api/client';
	import { p, route } from '../router';
	import { continueAfterSignIn } from '../utils/signin';

	const token = $derived(route.params.token ?? '');
	let error = $state('');

	$effect(() => {
		if (!token) return;
		verifyEmail(token)
			.then(continueAfterSignIn)
			.catch(async (err) => (error = await errorMessage(err, 'This confirmation link does not work.')));
	});
</script>

<CenteredScreen subtitle="Confirm your email address">
	{#if error}
		<p class="mb-4 text-center text-gray-300">{error}</p>
		<a href={p('/login')} class="block text-center text-sm text-primary-500 hover:underline">Go to sign in</a>
	{:else}
		<p class="text-center text-gray-400">Confirming…</p>
	{/if}
</CenteredScreen>
