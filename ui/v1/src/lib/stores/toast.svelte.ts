/** Short confirmations at the bottom of the screen ("Link moved"), optionally with one action such as Undo. */
export interface Toast {
	id: number;
	text: string;
	action?: { label: string; run: () => void };
}

export const toasts = $state<{ list: Toast[] }>({ list: [] });

let nextId = 1;

/** `durationMs` 0 keeps the toast until it is dismissed or its action runs. */
export function showToast(text: string, action?: Toast['action'], durationMs = 4000) {
	const toast = { id: nextId++, text, action };
	toasts.list = [...toasts.list.slice(-2), toast];
	if (durationMs > 0) setTimeout(() => dismissToast(toast.id), durationMs);
}

export function dismissToast(id: number) {
	toasts.list = toasts.list.filter((toast) => toast.id !== id);
}
