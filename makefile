# Ragbag: Go server (cmd/ragbag) with the Svelte frontend (ui/v1) embedded from ui/v1/dist.

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
# docker or podman, whichever is installed
CONTAINER ?= $(shell command -v docker 2>/dev/null || command -v podman 2>/dev/null)
OPENAPI_GENERATOR := docker.io/openapitools/openapi-generator-cli:v7.26.0

.PHONY: help build build-ui build-server build-linux install generate dev dev-ui test test-postgres check docker clean

help: ## Show this help
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-14s %s\n", $$1, $$2}'

build: build-ui build-server ## Build bin/ragbag with the frontend embedded

build-ui: ## Build the frontend into ui/v1/dist
	cd ui/v1 && npm ci --no-audit --no-fund && VITE_APP_BASE=/app npm run build

build-server: ## Build the Go binary (static, no CGO)
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/ragbag ./cmd/ragbag

build-linux: build-ui ## Build bin/ragbag-linux-{amd64,arm64} for servers (e.g. when building on a Mac)
	for arch in amd64 arm64; do \
		CGO_ENABLED=0 GOOS=linux GOARCH=$$arch go build -trimpath -ldflags "$(LDFLAGS)" -o bin/ragbag-linux-$$arch ./cmd/ragbag; \
	done

install: build ## Install/update as a systemd service (sudo); DIR=/opt/ragbag to choose the binary location
	sudo ./scripts/install.sh --binary bin/ragbag $(if $(DIR),--dir $(DIR))

generate: ## Regenerate server interfaces and the TypeScript client from openapi/openapi.yaml
	go generate ./internal/api
	$(CONTAINER) run --rm -v "$(CURDIR):/local:Z" $(OPENAPI_GENERATOR) generate \
		-i /local/openapi/openapi.yaml -g typescript-fetch -o /local/ui/v1/src/lib/sdk \
		--additional-properties=supportsES6=true,typescriptThreePlus=true,useSingleRequestParameter=true

dev: ## Run the server on :8080 with the settings from .env.local (demo account admin/test123)
	@test -f .env.local || (cp .env.local.example .env.local && echo "created .env.local from .env.local.example")
	go run ./cmd/ragbag

dev-ui: ## Run the Vite dev server on :5173 (talks to the server on :8080)
	cd ui/v1 && npm run dev

test: ## Run all Go tests against SQLite
	go test ./...

test-postgres: ## Run the tests against a throwaway PostgreSQL container
	$(CONTAINER) run -d --rm --name ragbag-test-db -e POSTGRES_PASSWORD=test -p 15432:5432 postgres:18-alpine
	@sleep 5
	RAGBAG_TEST_POSTGRES="postgres://postgres:test@localhost:15432/postgres?sslmode=disable" go test -count=1 ./...; \
		status=$$?; $(CONTAINER) rm -f ragbag-test-db >/dev/null; exit $$status

check: ## Lint the Go code and type-check the frontend
	go vet ./...
	test -z "$$(gofmt -l cmd internal ui)"
	cd ui/v1 && npm run check

docker: ## Build the container image ragbag:latest
	$(CONTAINER) build --build-arg VERSION=$(VERSION) -t ragbag:latest .

clean: ## Remove build output
	rm -rf bin
	find ui/v1/dist -mindepth 1 ! -name .gitkeep -delete
