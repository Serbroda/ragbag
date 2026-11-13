import { takePendingAdd, takePendingInvite } from '../api/client';
import { navigate } from '../router';
import { loadMe, resetSession } from '../stores/spaces.svelte';

/**
 * Continues where a signed-out visitor wanted to go (an invite, a link to add), else the start page; the setup
 * admin first replaces its initial password, everybody accepts changed terms of use.
 */
export async function continueAfterSignIn() {
	resetSession();
	const me = await loadMe().catch(() => null);
	if (me?.mustChangePassword) {
		navigate('/setup');
		return;
	}
	if (me?.termsPending) {
		navigate('/terms');
		return;
	}
	const pendingInvite = takePendingInvite();
	const pendingAdd = takePendingAdd();
	if (pendingInvite) {
		navigate('/invite/:token', { params: { token: pendingInvite } });
	} else if (pendingAdd !== null) {
		window.location.href = `/add${pendingAdd}`;
	} else {
		navigate('/');
	}
}
