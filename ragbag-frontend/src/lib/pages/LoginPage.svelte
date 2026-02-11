<script lang="ts">
	import { Card, Label, Input, Button, Alert } from 'flowbite-svelte';
	import { login, register } from '../api/client';
	import { navigate } from '../router';

	let isRegister = $state(false);
	let username = $state('');
	let email = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let error = $state('');
	let success = $state('');
	let loading = $state(false);

	function resetForm() {
		username = '';
		email = '';
		password = '';
		confirmPassword = '';
		error = '';
		success = '';
	}

	function toggleMode() {
		isRegister = !isRegister;
		resetForm();
	}

	async function handleSubmit(e: Event) {
		e.preventDefault();
		error = '';
		success = '';
		loading = true;

		try {
			if (isRegister) {
				if (password !== confirmPassword) {
					error = 'Passwords do not match.';
					loading = false;
					return;
				}
				await register(username, email, password);
				success = 'Registration successful! You can now sign in.';
				isRegister = false;
				username = '';
				email = '';
				password = '';
				confirmPassword = '';
			} else {
				await login(username, password);
				navigate('/');
			}
		} catch {
			error = isRegister
				? 'Registration failed. Please try again.'
				: 'Login failed. Please check your credentials.';
		} finally {
			loading = false;
		}
	}
</script>

<div class="flex min-h-screen items-center justify-center bg-gray-50 dark:bg-gray-900">
	<Card class="w-full max-w-md">
		<h2 class="mb-4 text-center text-2xl font-bold text-gray-900 dark:text-white">Ragbag</h2>
		<p class="mb-4 text-center text-sm text-gray-500 dark:text-gray-400">
			{isRegister ? 'Create a new account' : 'Sign in to your account'}
		</p>

		{#if error}
			<Alert color="red" class="mb-4">{error}</Alert>
		{/if}
		{#if success}
			<Alert color="green" class="mb-4">{success}</Alert>
		{/if}

		<form onsubmit={handleSubmit} class="flex flex-col gap-4">
			<div>
				<Label for="username" class="mb-2">Username</Label>
				<Input id="username" type="text" bind:value={username} required />
			</div>
			{#if isRegister}
				<div>
					<Label for="email" class="mb-2">Email</Label>
					<Input id="email" type="email" bind:value={email} required />
				</div>
			{/if}
			<div>
				<Label for="password" class="mb-2">Password</Label>
				<Input id="password" type="password" bind:value={password} required />
			</div>
			{#if isRegister}
				<div>
					<Label for="confirmPassword" class="mb-2">Confirm Password</Label>
					<Input id="confirmPassword" type="password" bind:value={confirmPassword} required />
				</div>
			{/if}
			<Button type="submit" disabled={loading}>
				{#if loading}
					{isRegister ? 'Creating account...' : 'Signing in...'}
				{:else}
					{isRegister ? 'Create account' : 'Sign in'}
				{/if}
			</Button>
		</form>

		<div class="mt-4 text-center text-sm text-gray-500 dark:text-gray-400">
			{#if isRegister}
				Already have an account?
				<button type="button" class="font-medium text-primary-600 hover:underline dark:text-primary-500" onclick={toggleMode}>
					Sign in
				</button>
			{:else}
				Don't have an account?
				<button type="button" class="font-medium text-primary-600 hover:underline dark:text-primary-500" onclick={toggleMode}>
					Register
				</button>
			{/if}
		</div>
	</Card>
</div>
