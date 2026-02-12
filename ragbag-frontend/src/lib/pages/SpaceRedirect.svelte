<script lang="ts">
	import { SpaceApi } from 'ragbag-frontend-sdk';
	import { apiConfig } from '../api/client';
	import { navigate } from '../router';

	new SpaceApi(apiConfig()).getSpaces().then((spaces) => {
		if (spaces.length > 0) {
			navigate('/space/:spaceId', { params: { spaceId: spaces[0].id } });
		}
	});
</script>

<div class="flex min-h-screen items-center justify-center bg-gray-50 dark:bg-gray-900">
	<p class="text-gray-500 dark:text-gray-400">Loading...</p>
</div>
