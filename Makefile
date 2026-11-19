SHELL := /bin/sh

APP       := simple-go-crud-app
BIN_DIR   := bin
BIN       := $(BIN_DIR)/$(APP)
DB_PATH   ?= persistence/db/app.db
ADDR      ?= :8080
# Origins allowed to call the API from a browser; Scalar docs runs on 8082.
CORS_ORIGINS ?= http://localhost:8082
# Development-only auth defaults. Production must provide JWT_SECRET and
# ADMIN_PASSWORD from the environment instead of these values.
JWT_SECRET       ?= dev-only-jwt-secret-change-me-32-bytes
ACCESS_TOKEN_TTL ?= 15m
REFRESH_TOKEN_TTL ?= 168h
ADMIN_USERNAME   ?= admin
ADMIN_PASSWORD   ?= admin12345
ADMIN_NAME       ?= Administrator
UPLOAD_MAX_SIZE  ?= 10MB
# Build goose with SQLite support only; the project does not use other drivers.
GOOSE_TAGS ?= no_clickhouse no_libsql no_mssql no_azuresql no_mysql no_postgres no_vertica no_ydb
GOOSE      ?= go run -tags '$(GOOSE_TAGS)' github.com/pressly/goose/v3/cmd/goose@v3.30.0
# API docs and request testing are rendered by Scalar via Node.js; the ports
# avoid the app (8080) and sqlite-web (8083).
SCALAR    ?= npx --yes @scalar/cli@2.17.0
DOCS_PORT ?= 8082
MOCK_PORT ?= 8081
GO_FILES  := $(shell find . -type f -name '*.go' -not -path './vendor/*')

.DEFAULT_GOAL := help

.PHONY: help run build test vet fmt fmt-check tidy check \
	api-validate docs mock \
	migrate-up migrate-down migrate-status migrate-create clean

help: ## Tampilkan daftar target
	@awk 'BEGIN {FS = ":.*## "; printf "\nPenggunaan:\n  make \033[36m<target>\033[0m\n\nTarget:\n"} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

run: ## Jalankan server HTTP (migrasi otomatis)
	CORS_ORIGINS=$(CORS_ORIGINS) ADDR=$(ADDR) DB_PATH=$(DB_PATH) \
	JWT_SECRET='$(JWT_SECRET)' ACCESS_TOKEN_TTL=$(ACCESS_TOKEN_TTL) \
	REFRESH_TOKEN_TTL=$(REFRESH_TOKEN_TTL) ADMIN_USERNAME='$(ADMIN_USERNAME)' \
	ADMIN_PASSWORD='$(ADMIN_PASSWORD)' ADMIN_NAME='$(ADMIN_NAME)' \
	UPLOAD_MAX_SIZE=$(UPLOAD_MAX_SIZE) go run .

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

api-validate: ## Validasi kontrak OpenAPI dengan Scalar (butuh Node.js)
	$(SCALAR) document validate api/openapi.yaml

docs: ## Preview kontrak + coba request di browser (butuh Node.js, port 8082)
	$(SCALAR) document serve api/openapi.yaml --config api/scalar.config.json --port $(DOCS_PORT) --watch

mock: ## Mock server dari kontrak, untuk FE sebelum BE ada (port 8081)
	$(SCALAR) document mock api/openapi.yaml --port $(MOCK_PORT)

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
