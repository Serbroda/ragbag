SHELL := /bin/bash

# ------------------------------------------------------------
# Project informations
# ------------------------------------------------------------
BINARY_NAME := ragbag
BINARY_VERSION := 0.1.0

# Paths
OUT_DIR := bin
SERVER_MAIN_DIR := ./cmd/server


# ------------------------------------------------------------
# Default targets
# ------------------------------------------------------------
.PHONY: all build generate-go clean test

all: build

build: clean generate-go
	@echo "==> Building Go binaries for platforms..."
	$(call build_bin,darwin,amd64,macos-amd64)
	$(call build_bin,darwin,arm64,macos-arm64)
	$(call build_bin,linux,amd64,linux-amd64)
	$(call build_bin,windows,amd64,windows-amd64.exe)
	@echo "==> Build complete!"

define build_bin
	@echo "==> Building Go binary for $(1)/$(2)..."
	GOOS=$(1) GOARCH=$(2) CGO_ENABLED=0 \
		go build -ldflags "-X main.Version=$(BINARY_VERSION)" -o ${OUT_DIR}/${BINARY_NAME}-v${BINARY_VERSION}-$(3) ${SERVER_MAIN_DIR}
endef

generate-go:
	@echo "==> Generating Go code..."
	go generate ./...
	@echo "==> Generation done."

clean:
	@echo "==> Cleaning up..."
	rm -rf bin/

test:
	@echo "==> Running tests..."
	go test ./... -v


# ------------------------------------------------------------
# Docker targets
# ------------------------------------------------------------
# TBD
