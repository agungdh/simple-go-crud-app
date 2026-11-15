SHELL := /bin/sh

APP       := simple-go-crud-app
BIN_DIR   := bin
BIN       := $(BIN_DIR)/$(APP)
DB_PATH   ?= persistence/db/app.db
# Build goose with SQLite support only; the project does not use other drivers.
GOOSE_TAGS ?= no_clickhouse no_libsql no_mssql no_azuresql no_mysql no_postgres no_vertica no_ydb
GOOSE      ?= go run -tags '$(GOOSE_TAGS)' github.com/pressly/goose/v3/cmd/goose@v3.30.0
# API docs are rendered by Redoc via Node.js; the port avoids the app (8080)
# and sqlite-web (8083).
REDOCLY   ?= npx --yes @redocly/cli@1.34.24
DOCS_PORT ?= 8082
GO_FILES  := $(shell find . -type f -name '*.go' -not -path './vendor/*')

.DEFAULT_GOAL := help

.PHONY: help run build test vet fmt fmt-check tidy check \
	api-validate docs \
	migrate-up migrate-down migrate-status migrate-create clean

help: ## Tampilkan daftar target
	@awk 'BEGIN {FS = ":.*## "; printf "\nPenggunaan:\n  make \033[36m<target>\033[0m\n\nTarget:\n"} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

run: ## Buka database + migration lalu keluar (belum ada server)
	DB_PATH=$(DB_PATH) go run .

build: ## Build binary ke bin/
	@mkdir -p $(BIN_DIR)
	go build -trimpath -o $(BIN) .

test: ## Jalankan semua test
	go test ./...

vet: ## Jalankan go vet
	go vet ./...

fmt: ## Format semua file Go
	gofmt -w $(GO_FILES)

fmt-check: ## Cek format tanpa mengubah file
	@files="$$(gofmt -l $(GO_FILES))"; \
	if [ -n "$$files" ]; then \
		echo "File belum diformat:"; \
		echo "$$files"; \
		exit 1; \
	fi

tidy: ## Rapikan go.mod dan go.sum
	go mod tidy

check: fmt-check vet test ## Jalankan fmt-check, vet, dan test

api-validate: ## Validasi kontrak OpenAPI dan kecocokan DTO
	go test ./api ./internal/dto

docs: ## Preview kontrak OpenAPI di browser (butuh Node.js, port 8082)
	$(REDOCLY) preview-docs api/openapi.yaml -p $(DOCS_PORT)

migrate-up: ## Jalankan semua migration
	$(GOOSE) -dir migrations sqlite3 $(DB_PATH) up

migrate-down: ## Rollback satu migration terakhir
	$(GOOSE) -dir migrations sqlite3 $(DB_PATH) down

migrate-status: ## Tampilkan status migration
	$(GOOSE) -dir migrations sqlite3 $(DB_PATH) status

migrate-create: ## Buat migration baru (NAME=nama_migration)
	@if [ -z "$(NAME)" ]; then \
		echo "Gunakan: make migrate-create NAME=nama_migration"; \
		exit 1; \
	fi
	$(GOOSE) -dir migrations sqlite3 $(DB_PATH) create $(NAME) sql

clean: ## Hapus hasil build
	rm -rf $(BIN_DIR)
