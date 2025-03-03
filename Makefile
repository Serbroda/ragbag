SHELL := /bin/bash

PROJECT_NAME     := github.com/Serbroda/ragbag

SQLC_CONFIG_DIR := internal/db/sqlc
SQLC_GEN_OUTPUT_DIR := internal/db/sqlc/gen

generate-go:
	@echo "==> Generating Go code..."
	go generate ./...
	@echo "==> Generation done."
