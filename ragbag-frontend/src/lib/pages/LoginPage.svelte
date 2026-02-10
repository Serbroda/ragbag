<script lang="ts">
	import { Card, Label, Input, Button, Alert } from 'flowbite-svelte';
	import { login } from '../api/client';
	import { navigate } from '../router';

	let username = $state('');
	let password = $state('');
	let error = $state('');
	let loading = $state(false);

	async function handleSubmit(e: Event) {
		e.preventDefault();
		error = '';
		loading = true;
		try {
			await login(username, password);
			navigate('/');
		} catch {
			error = 'Login failed. Please check your credentials.';
		} finally {
			loading = false;
		}
	}
</script>

<div class="flex min-h-screen items-center justify-center bg-gray-50 dark:bg-gray-900">
	<Card class="w-full max-w-md">
		<h2 class="mb-4 text-center text-2xl font-bold text-gray-900 dark:text-white">Ragbag</h2>

		{#if error}
			<Alert color="red" class="mb-4">{error}</Alert>
		{/if}

		<form onsubmit={handleSubmit} class="flex flex-col gap-4">
			<div>
				<Label for="username" class="mb-2">Username</Label>
				<Input id="username" type="text" bind:value={username} required />
			</div>
			<div>
				<Label for="password" class="mb-2">Password</Label>
				<Input id="password" type="password" bind:value={password} required />
			</div>
			<Button type="submit" disabled={loading}>
				{loading ? 'Signing in...' : 'Sign in'}
			</Button>
		</form>
	</Card>
</div>
