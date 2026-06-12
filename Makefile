GO ?= go
CARGO ?= cargo
RUSTUP ?= rustup
ATLAS_VERSION ?= 0.30.0
SQLC_VERSION ?= 1.31.0
CACHE_ROOT ?= /tmp/ai-arena-go-quality-gates
GOPATH = $(CACHE_ROOT)/go
GOMODCACHE = $(GOPATH)/pkg/mod
GOCACHE = $(CACHE_ROOT)/go-build
CARGO_TARGET_DIR ?= $(CACHE_ROOT)/cargo-target
RUST_WASM_TARGET ?= wasm32-wasip1
AI_ARENA_PG_TEST_DSN ?= postgres://arena:arena@127.0.0.1:55432/arena_service?sslmode=disable
AI_ARENA_PG_MIGRATION_DSN ?= $(AI_ARENA_PG_TEST_DSN)
AI_ARENA_PG_ATLAS_DEV_DSN ?= postgres://arena:arena@127.0.0.1:55432/postgres?sslmode=disable
SEAWEED_DATA_DIR ?= $(CURDIR)/.local/seaweed
SEAWEED_COMPOSE_FILE ?= tools/dev/seaweed-compose.yml
SEAWEED_BUCKET ?= ai-arena-local
SEAWEED_ENDPOINT ?= http://127.0.0.1:8333
AWS_CLI_IMAGE ?= public.ecr.aws/aws-cli/aws-cli:latest
ATLAS_IMAGE ?= arigaio/atlas:$(ATLAS_VERSION)
SQLC_IMAGE ?= sqlc/sqlc:$(SQLC_VERSION)
ATLAS_DOCKER = docker run --rm --network host -v "$(CURDIR):/work" -w /work -v /var/run/docker.sock:/var/run/docker.sock $(ATLAS_IMAGE)
SQLC_DOCKER = docker run --rm -u "$$(id -u):$$(id -g)" -v "$(CURDIR):/work" -w /work $(SQLC_IMAGE)
POSTGRES_SCHEMA_DIR ?= internal/platform/service/postgres/schema
POSTGRES_MIGRATIONS_DIR ?= internal/platform/service/postgres/migrations
POSTGRES_SQLC_CONFIG ?= sqlc.yaml
POSTGRES_ATLAS_DEV_URL ?= $(AI_ARENA_PG_ATLAS_DEV_DSN)
POSTGRES_MIGRATION_NAME ?=
POSTGRES_MIGRATION_VERSION ?=
POSTGRES_MIGRATION_REVISIONS_SCHEMA ?=
POSTGRES_SCHEMA_URL ?= file://$(POSTGRES_SCHEMA_DIR)
POSTGRES_MIGRATIONS_URL ?= file://$(POSTGRES_MIGRATIONS_DIR)
GO_ENV = GOPATH=$(GOPATH) GOMODCACHE=$(GOMODCACHE) GOCACHE=$(GOCACHE)
GOFILES = $(shell git ls-files -- '*.go' | while read -r file; do if [ -f "$$file" ]; then printf '%s ' "$$file"; fi; done)
REVIVE_TESTDATA_DIRS = $(shell git ls-files -- testdata internal/platform/runtime/testdata | while read -r file; do if [ -f "$$file" ] && printf '%s' "$$file" | grep -q '\.go$$'; then dirname "$$file"; fi; done | sort -u | tr '\n' ' ')
REVIVE_SOURCE_PATTERNS = $(shell for dir in cmd games internal e2e; do if [ -d "$$dir" ]; then printf './%s/... ' "$$dir"; fi; done)
REVIVE_PACKAGE_DIRS = $(shell $(GO) list -f '{{.Dir}}' $(REVIVE_SOURCE_PATTERNS) | grep -v '/internal/platform/service/postgres/sqlc$$' | tr '\n' ' ')

.PHONY: test test-postgres postgres-up postgres-down postgres-schema-apply postgres-migrate-diff postgres-migrate-hash postgres-migrate-baseline postgres-migrate-apply postgres-sqlc-generate seaweed-up seaweed-down seaweed-bootstrap verify-local-object-storage test-wasm-go test-wasm-rust fmt lint lint-goimports lint-vet lint-noctx lint-staticcheck lint-gosec lint-revive build-preset-bots render-build render-start build-janken-go-wasm run-janken-go-wasm build-janken-rust-wasm run-janken-rust-wasm-eval run-echo-simultaneous run-echo-sequential

export COMPOSE_BAKE = false

test:
	mkdir -p "$(GOPATH)" "$(GOCACHE)" "$(GOMODCACHE)"
	$(GO_ENV) $(GO) test ./...

test-postgres:
	mkdir -p "$(GOPATH)" "$(GOCACHE)" "$(GOMODCACHE)"
	$(MAKE) postgres-schema-apply
	AI_ARENA_PG_TEST_DSN="$(AI_ARENA_PG_TEST_DSN)" $(GO_ENV) $(GO) test ./...

postgres-up:
	docker compose -f tools/dev/postgres-compose.yml up -d postgres

postgres-down:
	docker compose -f tools/dev/postgres-compose.yml down -v

seaweed-up:
	mkdir -p "$(SEAWEED_DATA_DIR)"
	SEAWEED_DATA_DIR="$(SEAWEED_DATA_DIR)" docker compose -f "$(SEAWEED_COMPOSE_FILE)" up -d seaweed

seaweed-down:
	SEAWEED_DATA_DIR="$(SEAWEED_DATA_DIR)" docker compose -f "$(SEAWEED_COMPOSE_FILE)" down -v

seaweed-bootstrap:
	SEAWEED_DATA_DIR="$(SEAWEED_DATA_DIR)" SEAWEED_BUCKET="$(SEAWEED_BUCKET)" SEAWEED_ENDPOINT="$(SEAWEED_ENDPOINT)" AWS_CLI_IMAGE="$(AWS_CLI_IMAGE)" ./tools/dev/seaweed-bootstrap.sh

verify-local-object-storage:
	mkdir -p "$(GOPATH)" "$(GOCACHE)" "$(GOMODCACHE)"
	ARENA_SERVICE_BASE_URL="http://127.0.0.1:$${PORT:-10000}" $(GO_ENV) $(GO) run ./tools/dev/verify-local-object-storage.go

postgres-schema-apply:
	@attempt=0; \
	until [ "$$attempt" -ge 10 ]; do \
		if $(ATLAS_DOCKER) schema apply --auto-approve --url "$(AI_ARENA_PG_TEST_DSN)" --to "$(POSTGRES_SCHEMA_URL)" --dev-url "$(POSTGRES_ATLAS_DEV_URL)"; then \
			exit 0; \
		fi; \
		attempt="$$((attempt + 1))"; \
		echo "postgres-schema-apply attempt $$attempt failed; retrying in 2s" >&2; \
		sleep 2; \
	done; \
	echo "postgres-schema-apply failed after $$attempt attempts"; \
	exit 1

postgres-migrate-diff:
	@if [ -z "$(NAME)" ] && [ -z "$(POSTGRES_MIGRATION_NAME)" ]; then \
		echo "NAME or POSTGRES_MIGRATION_NAME is required"; \
		exit 1; \
	fi
	name="$${NAME:-$(POSTGRES_MIGRATION_NAME)}"; \
	$(ATLAS_DOCKER) migrate diff "$$name" --dir "$(POSTGRES_MIGRATIONS_URL)" --to "$(POSTGRES_SCHEMA_URL)" --dev-url "$(POSTGRES_ATLAS_DEV_URL)"; \
	$(MAKE) postgres-migrate-hash

postgres-migrate-hash:
	$(ATLAS_DOCKER) migrate hash --dir "$(POSTGRES_MIGRATIONS_URL)"

postgres-migrate-baseline:
	@if [ -z "$(VERSION)" ] && [ -z "$(POSTGRES_MIGRATION_VERSION)" ]; then \
		echo "VERSION or POSTGRES_MIGRATION_VERSION is required"; \
		exit 1; \
	fi
	version="$${VERSION:-$(POSTGRES_MIGRATION_VERSION)}"; \
	ATLAS_IMAGE="$(ATLAS_IMAGE)" POSTGRES_MIGRATIONS_URL="$(POSTGRES_MIGRATIONS_URL)" AI_ARENA_PG_MIGRATION_DSN="$(AI_ARENA_PG_MIGRATION_DSN)" POSTGRES_MIGRATION_BASELINE_VERSION="$$version" POSTGRES_MIGRATION_REVISIONS_SCHEMA="$(POSTGRES_MIGRATION_REVISIONS_SCHEMA)" ./tools/dev/postgres-migrate-apply.sh

postgres-migrate-apply:
	ATLAS_IMAGE="$(ATLAS_IMAGE)" POSTGRES_MIGRATIONS_URL="$(POSTGRES_MIGRATIONS_URL)" AI_ARENA_PG_MIGRATION_DSN="$(AI_ARENA_PG_MIGRATION_DSN)" POSTGRES_MIGRATION_BASELINE_VERSION="$(POSTGRES_MIGRATION_BASELINE_VERSION)" POSTGRES_MIGRATION_REVISIONS_SCHEMA="$(POSTGRES_MIGRATION_REVISIONS_SCHEMA)" ./tools/dev/postgres-migrate-apply.sh

postgres-sqlc-generate:
	$(SQLC_DOCKER) generate -f "$(POSTGRES_SQLC_CONFIG)"

test-wasm-go:
	mkdir -p "$(GOPATH)" "$(GOCACHE)" "$(GOMODCACHE)"
	AI_ARENA_WASM_E2E=1 $(GO_ENV) $(GO) test ./e2e -run '^(TestArenaRunnerJankenGoWASMMixedRuntimePath|TestArenaRunnerJankenGoWASMMissingModuleFails|TestBuildGoWASMReportsBuildFailure)$$'

test-wasm-rust:
	mkdir -p "$(GOPATH)" "$(GOCACHE)" "$(GOMODCACHE)" "$(CARGO_TARGET_DIR)"
	AI_ARENA_WASM_E2E=1 AI_ARENA_EXPERIMENT_RUST_WASM=1 $(GO_ENV) CARGO_TARGET_DIR="$(CARGO_TARGET_DIR)" $(GO) test ./e2e -run '^TestArenaRunnerJankenRustWASMEvaluationPath$$'

fmt:
	mkdir -p "$(GOPATH)" "$(GOCACHE)" "$(GOMODCACHE)"
	if [ -n "$(GOFILES)" ]; then $(GO_ENV) $(GO) tool goimports -w $(GOFILES); fi

lint: lint-goimports lint-vet lint-noctx lint-staticcheck lint-gosec lint-revive

lint-goimports:
	mkdir -p "$(GOPATH)" "$(GOCACHE)" "$(GOMODCACHE)"
	if [ -n "$(GOFILES)" ]; then \
		output="$$( $(GO_ENV) $(GO) tool goimports -l $(GOFILES) )"; \
		if [ -n "$$output" ]; then \
			printf '%s\n' "$$output"; \
			exit 1; \
		fi; \
	fi

lint-vet:
	mkdir -p "$(GOPATH)" "$(GOCACHE)" "$(GOMODCACHE)"
	$(GO_ENV) $(GO) vet ./...

lint-noctx:
	mkdir -p "$(GOPATH)" "$(GOCACHE)" "$(GOMODCACHE)"
	noctx_bin="$$( $(GO_ENV) $(GO) tool -n noctx )"; \
	$(GO_ENV) $(GO) vet -vettool="$$noctx_bin" ./...

lint-staticcheck:
	mkdir -p "$(GOPATH)" "$(GOCACHE)" "$(GOMODCACHE)"
	$(GO_ENV) $(GO) tool staticcheck ./...

lint-gosec:
	mkdir -p "$(GOPATH)" "$(GOCACHE)" "$(GOMODCACHE)"
	$(GO_ENV) $(GO) tool gosec -exclude-dir=.cache ./...

lint-revive:
	mkdir -p "$(GOPATH)" "$(GOCACHE)" "$(GOMODCACHE)"
	$(GO_ENV) $(GO) tool revive -config revive.toml $(REVIVE_PACKAGE_DIRS) $(REVIVE_TESTDATA_DIRS)

build-preset-bots:
	mkdir -p "$(GOPATH)" "$(GOCACHE)" "$(GOMODCACHE)"
	$(GO_ENV) ./tools/dev/build-preset-bots.sh

render-build:
	mkdir -p "$(GOPATH)" "$(GOCACHE)" "$(GOMODCACHE)"
	$(MAKE) build-preset-bots
	$(GO_ENV) $(GO) build -tags netgo -ldflags '-s -w' -o app ./cmd/arena-service

render-start:
	./app serve \
		--listen-addr "0.0.0.0:$${PORT:-10000}" \
		--preset-config "$${ARENA_SERVICE_PRESET_CONFIG:-./config/platform-service/presets.remote-bootstrap.json}"

build-janken-go-wasm:
	mkdir -p "$(GOPATH)" "$(GOCACHE)" "$(GOMODCACHE)"
	$(GO_ENV) GOOS=wasip1 GOARCH=wasm $(GO) build -o ./testdata/ai/janken/janken-go-wasm-ai.wasm ./testdata/ai/janken/janken-go-wasm-ai

run-janken-go-wasm: build-janken-go-wasm
	mkdir -p "$(GOCACHE)" "$(GOMODCACHE)"
	output_dir="$$(mktemp -d /tmp/ai-arena-janken-wasm-XXXXXX)"; \
	echo "artifact dir: $$output_dir/janken-go-wasm"; \
	$(GO_ENV) $(GO) run ./cmd/arena-runner \
		--game janken \
		--game-version 2.1.0 \
		--ruleset regular \
		--match-id janken-go-wasm \
		--output-dir "$$output_dir" \
		--player p1=./testdata/ai/janken/janken-go-wasm-ai \
		--player p2=./testdata/ai/janken/janken-rock-ai-wasm

build-janken-rust-wasm:
	mkdir -p "$(CARGO_TARGET_DIR)"
	@if ! $(RUSTUP) target list --installed | grep -qx "$(RUST_WASM_TARGET)"; then \
		echo "missing Rust target $(RUST_WASM_TARGET); run: $(RUSTUP) target add $(RUST_WASM_TARGET)"; \
		exit 1; \
	fi
	CARGO_TARGET_DIR="$(CARGO_TARGET_DIR)" $(CARGO) build \
		--manifest-path ./testdata/ai/janken/janken-rust-wasm-ai/Cargo.toml \
		--target $(RUST_WASM_TARGET) \
		--release
	cp "$(CARGO_TARGET_DIR)/$(RUST_WASM_TARGET)/release/janken-rust-wasm-ai.wasm" ./testdata/ai/janken/janken-rust-wasm-ai.wasm

run-janken-rust-wasm-eval: build-janken-rust-wasm
	mkdir -p "$(GOCACHE)" "$(GOMODCACHE)"
	output_dir="$$(mktemp -d /tmp/ai-arena-janken-rust-wasm-XXXXXX)"; \
	echo "artifact dir: $$output_dir/janken-rust-wasm-eval"; \
	$(GO_ENV) $(GO) run ./cmd/arena-runner \
		--game janken \
		--game-version 2.1.0 \
		--ruleset regular \
		--match-id janken-rust-wasm-eval \
		--output-dir "$$output_dir" \
		--player p1=./testdata/ai/janken/janken-rust-wasm-ai \
		--player p2=./testdata/ai/janken/janken-rock-ai-wasm

run-echo-simultaneous:
	mkdir -p "$(GOCACHE)" "$(GOMODCACHE)"
	output_dir="$$(mktemp -d /tmp/ai-arena-sim-XXXXXX)"; \
	echo "artifact dir: $$output_dir/sim-happy"; \
	$(GO_ENV) $(GO) run ./cmd/arena-runner \
		--game echo-count \
		--game-version 2.0.0 \
		--ruleset phase2-simultaneous-3turn \
		--match-id sim-happy \
		--output-dir "$$output_dir" \
		--player p1=./testdata/ai/echo/echo-ai \
		--player p2=./testdata/ai/echo/echo-ai

run-echo-sequential:
	mkdir -p "$(GOCACHE)" "$(GOMODCACHE)"
	output_dir="$$(mktemp -d /tmp/ai-arena-seq-XXXXXX)"; \
	echo "artifact dir: $$output_dir/seq-happy"; \
	$(GO_ENV) $(GO) run ./cmd/arena-runner \
		--game echo-count \
		--game-version 2.0.0 \
		--ruleset phase2-sequential-3turn \
		--match-id seq-happy \
		--output-dir "$$output_dir" \
		--player p1=./testdata/ai/echo/echo-ai-sequential \
		--player p2=./testdata/ai/echo/echo-ai-sequential
