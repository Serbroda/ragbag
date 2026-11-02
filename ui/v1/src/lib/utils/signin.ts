import { appBase, takePendingAdd, takePendingInvite } from '../api/client';
import { navigate } from '../router';
import { resetSession } from '../stores/spaces.svelte';

/** Continues where a signed-out visitor wanted to go (an invite, a link to add), else the start page. */
export function continueAfterSignIn() {
	resetSession();
	const pendingInvite = takePendingInvite();
	const pendingAdd = takePendingAdd();
	if (pendingInvite) {
		navigate('/invite/:token', { params: { token: pendingInvite } });
	} else if (pendingAdd !== null) {
		window.location.href = `${appBase()}/add${pendingAdd}`;
	} else {
		navigate('/');
	}
}
