<script lang="ts">
	import { Button, Input, Label, Modal, Textarea } from 'flowbite-svelte';
	import type { ReportCategory } from 'ragbag-frontend-sdk';
	import { errorMessage, reportShare } from '../api/client';

	/** Reports what a public link shows to the administrators of this Ragbag (no account needed). */
	let {
		open = $bindable(false),
		token,
		collectionId,
		collectionName,
	}: { open?: boolean; token: string; collectionId?: string; collectionName: string } = $props();

	const REPORT_CATEGORIES: { value: ReportCategory; label: string }[] = [
		{ value: 'ILLEGAL', label: 'Illegal content' },
		{ value: 'CHILD_SAFETY', label: 'Child sexual abuse material' },
		{ value: 'COPYRIGHT', label: 'Copyright or trademark infringement' },
		{ value: 'PERSONAL_DATA', label: 'Personal data shared without consent' },
		{ value: 'SPAM', label: 'Spam, scam or malware' },
		{ value: 'OTHER', label: 'Something else' },
	];

	let category = $state<ReportCategory>('ILLEGAL');
	let details = $state('');
	let name = $state('');
	let email = $state('');
	let goodFaith = $state(false);
	let busy = $state(false);
	let error = $state('');
	let sent = $state(false);

	$effect(() => {
		if (open) return;
		details = name = email = error = '';
		goodFaith = sent = false;
		category = 'ILLEGAL';
	});

	async function submit(e: Event) {
		e.preventDefault();
		busy = true;
		error = '';
		try {
			await reportShare(token, {
				category,
				collectionId,
				details,
				// Reports of abuse material may stay anonymous
				name: (category !== 'CHILD_SAFETY' && name.trim()) || undefined,
				email: (category !== 'CHILD_SAFETY' && email.trim()) || undefined,
				goodFaith,
			});
			sent = true;
		} catch (err) {
			error = await errorMessage(err, 'The report could not be sent.');
		} finally {
			busy = false;
		}
	}
</script>

<Modal bind:open size="sm" title="Report “{collectionName}”">
	{#if sent}
		<p class="text-sm text-gray-600 dark:text-gray-300">
			Thank you. The administrators of this Ragbag will look at your report{email.trim()
				? ' and let you know what they decided'
				: ''}.
		</p>
		<div class="flex justify-end"><Button onclick={() => (open = false)}>Close</Button></div>
	{:else}
		<form class="flex flex-col gap-4" onsubmit={submit}>
			<p class="text-sm text-gray-500 dark:text-gray-400">
				Tell the administrators of this Ragbag what is wrong with this collection. They can disable the public
				link.
			</p>
			<div>
				<Label for="report-category" class="mb-2">What is the problem?</Label>
				<select
					id="report-category"
					class="w-full rounded-lg border border-gray-300 bg-gray-50 p-2.5 text-sm text-gray-900 focus:border-primary-500 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white"
					bind:value={category}
				>
					{#each REPORT_CATEGORIES as option (option.value)}
						<option value={option.value}>{option.label}</option>
					{/each}
				</select>
			</div>
			<div>
				<Label for="report-details" class="mb-2">Details</Label>
				<Textarea
					id="report-details"
					rows={4}
					class="w-full"
					minlength={10}
					maxlength={5000}
					placeholder="Which links are affected and why, e.g. which law they break"
					bind:value={details}
					required
				/>
			</div>
			{#if category !== 'CHILD_SAFETY'}
				<div class="grid gap-3 sm:grid-cols-2">
					<div>
						<Label for="report-name" class="mb-2">Your name</Label>
						<Input id="report-name" autocomplete="name" maxlength={200} bind:value={name} />
					</div>
					<div>
						<Label for="report-email" class="mb-2">Your email address</Label>
						<Input id="report-email" type="email" autocomplete="email" maxlength={250} bind:value={email} />
					</div>
				</div>
				<p class="-mt-2 text-xs text-gray-500 dark:text-gray-400">
					Optional, to hear about the decision. Only the administrators see them.
				</p>
			{/if}
			<label class="flex items-start gap-2 text-sm text-gray-600 dark:text-gray-300">
				<input
					type="checkbox"
					class="mt-0.5 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-gray-600 dark:bg-gray-700"
					bind:checked={goodFaith}
				/>
				<span>I believe in good faith that the information in this report is accurate and complete.</span>
			</label>
			{#if error}
				<p class="text-sm text-red-500 dark:text-red-400">{error}</p>
			{/if}
			<div class="flex justify-end gap-2">
				<Button color="light" onclick={() => (open = false)}>Cancel</Button>
				<Button type="submit" color="red" disabled={busy || !goodFaith || details.trim().length < 10}>
					{busy ? 'Sending…' : 'Send report'}
				</Button>
			</div>
		</form>
	{/if}
</Modal>
