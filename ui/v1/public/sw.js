// Only answers page loads that fail with a notice, instead of the browser's error page. It deliberately caches
// nothing: no private links on the device, and a new version of the app is live with the next page load.
// Assets are cached by the browser already (immutable), the API is passed through untouched.

const OFFLINE_PAGE = `<!doctype html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta name="color-scheme" content="dark">
<title>Ragbag is offline</title>
<style>
body { margin: 0; min-height: 100dvh; display: flex; flex-direction: column; align-items: center; justify-content: center;
	padding: 1.5rem; box-sizing: border-box; text-align: center; background: #211d1b; color: #d6d3d1;
	font: 16px/1.5 ui-sans-serif, system-ui, -apple-system, 'Segoe UI', sans-serif; }
h1 { margin: 0 0 .5rem; font-size: 1.25rem; color: #fff; }
p { margin: 0 0 1.5rem; }
a { padding: .625rem 1.25rem; border-radius: .5rem; background: #d97706; color: #fff; font-weight: 500; text-decoration: none; }
</style>
</head>
<body>
<h1>Ragbag can't be reached</h1>
<p>You or the server seem to be offline.</p>
<a href="">Try again</a>
</body>
</html>`;

function offlinePage() {
	return new Response(OFFLINE_PAGE, {
		status: 503,
		headers: {
			'Content-Type': 'text/html; charset=utf-8',
			'Content-Security-Policy': "default-src 'none'; style-src 'unsafe-inline'; frame-ancestors 'none'",
			'Cache-Control': 'no-store',
		},
	});
}

self.addEventListener('install', () => self.skipWaiting());
self.addEventListener('activate', (event) => event.waitUntil(self.clients.claim()));

self.addEventListener('fetch', (event) => {
	if (event.request.mode !== 'navigate') return;
	event.respondWith(fetch(event.request).catch(offlinePage));
});
