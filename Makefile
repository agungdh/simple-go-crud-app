APP     := adminlte-demo
BIN_DIR := bin
BINARY  := $(BIN_DIR)/$(APP)
GO      ?= go
ARGS    ?=

.DEFAULT_GOAL := help
.PHONY: help build run build-prod clean

help: ## Tampilkan daftar target
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build binary development ke bin/
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BINARY) .

run: ## Jalankan aplikasi (contoh: make run ARGS="-addr :3000")
	$(GO) run . $(ARGS)

build-prod: ## Build binary statis + stripped untuk distribusi
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags="-s -w" -o $(BINARY) .

clean: ## Hapus hasil build
	rm -rf $(BIN_DIR)
