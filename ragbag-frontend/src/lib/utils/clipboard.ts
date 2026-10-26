/**
 * Copies text to the clipboard. `navigator.clipboard` only exists in secure contexts (HTTPS or localhost),
 * so self-hosted instances on plain http://192.168.x.x fall back to the legacy copy command.
 * Returns false if neither works; callers should then let the user copy the text manually.
 */
export async function copyText(text: string): Promise<boolean> {
	if (window.isSecureContext && navigator.clipboard) {
		try {
			await navigator.clipboard.writeText(text);
			return true;
		} catch {
			// Permission denied or document not focused: try the fallback
		}
	}
	const textarea = document.createElement('textarea');
	textarea.value = text;
	textarea.setAttribute('readonly', '');
	textarea.style.position = 'fixed';
	textarea.style.opacity = '0';
	// Inside an open modal the focus trap would steal focus from an element appended to <body>
	(document.activeElement?.closest('dialog, [role=dialog]') ?? document.body).appendChild(textarea);
	textarea.select();
	try {
		return document.execCommand('copy');
	} catch {
		return false;
	} finally {
		textarea.remove();
	}
}
