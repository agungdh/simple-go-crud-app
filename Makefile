APP     := adminlte-demo
BIN_DIR := bin
BINARY  := $(BIN_DIR)/$(APP)
GO      ?= go
ARGS    ?=
DB      ?= app.db
GOOSE   ?= goose

.DEFAULT_GOAL := help
.PHONY: help build run build-prod test vet migrate migrate-status clean

help: ## Tampilkan daftar target
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  %-16s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build binary development ke bin/
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BINARY) .

run: ## Jalankan aplikasi (contoh: make run ARGS="-addr :3000 -db app.db")
	$(GO) run . $(ARGS)

build-prod: ## Build binary statis + stripped untuk distribusi
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags="-s -w" -o $(BINARY) .

test: ## Jalankan seluruh test
	$(GO) test ./...

vet: ## Jalankan go vet
	$(GO) vet ./...

migrate: ## Terapkan migration via goose CLI (opsional; app juga auto-migrate)
	$(GOOSE) -dir internal/database/migrations sqlite3 "$(DB)" up

migrate-status: ## Lihat status migration via goose CLI
	$(GOOSE) -dir internal/database/migrations sqlite3 "$(DB)" status

clean: ## Hapus hasil build dan database runtime
	rm -rf $(BIN_DIR)
	rm -f "$(DB)" "$(DB)-shm" "$(DB)-wal"
