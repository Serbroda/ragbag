# Ragbag frontend (v1)

Svelte 5 + Vite. The build in `dist/` is embedded into the Go binary (`ui/ui.go`) and served under `/app`.
Setup, development and the API client are described in the [main README](../../README.md).

```bash
npm run dev     # hot reload on :5173, talks to the server on :8080 (make dev)
npm run check   # svelte-check + TypeScript
npm run build   # → dist/ (make build-ui sets VITE_APP_BASE=/app)
```

`src/lib/sdk/` is generated from `openapi/openapi.yaml` by `make generate` – do not edit it by hand.
