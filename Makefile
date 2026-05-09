GO ?= go
CARGO ?= cargo
RUSTUP ?= rustup
CACHE_ROOT ?= /tmp/ai-arena-go-quality-gates
DUNGEON_RULESET ?= seeded-maze-v1
DUNGEON_RNG_SEED ?= 0000000000000000000000000000000000000000000000000000000000000000
GOPATH = $(CACHE_ROOT)/go
GOMODCACHE = $(GOPATH)/pkg/mod
GOCACHE = $(CACHE_ROOT)/go-build
CARGO_TARGET_DIR ?= $(CACHE_ROOT)/cargo-target
RUST_WASM_TARGET ?= wasm32-wasip1
GO_ENV = GOPATH=$(GOPATH) GOMODCACHE=$(GOMODCACHE) GOCACHE=$(GOCACHE)
GOFILES = $(shell git ls-files -- '*.go')

.PHONY: test test-wasm-go test-wasm-rust fmt lint lint-goimports lint-vet lint-noctx lint-staticcheck lint-gosec build-janken-go-wasm run-janken-go-wasm build-janken-rust-wasm run-janken-rust-wasm-eval build-dungeon-go-wasm run-echo-simultaneous run-echo-sequential run-dungeon-local run-dungeon-local-quiet run-dungeon-go-wasm inspect-dungeon-map

test:
	mkdir -p "$(GOPATH)" "$(GOCACHE)" "$(GOMODCACHE)"
	$(GO_ENV) $(GO) test ./...

test-wasm-go:
	mkdir -p "$(GOPATH)" "$(GOCACHE)" "$(GOMODCACHE)"
	AI_ARENA_WASM_E2E=1 $(GO_ENV) $(GO) test ./e2e -run '^(TestArenaRunnerJankenGoWASMMixedRuntimePath|TestArenaRunnerJankenGoWASMMissingModuleFails|TestBuildGoWASMReportsBuildFailure|TestArenaRunnerDungeonGoWASMMixedRuntimePath)$$'

test-wasm-rust:
	mkdir -p "$(GOPATH)" "$(GOCACHE)" "$(GOMODCACHE)" "$(CARGO_TARGET_DIR)"
	AI_ARENA_WASM_E2E=1 AI_ARENA_EXPERIMENT_RUST_WASM=1 $(GO_ENV) CARGO_TARGET_DIR="$(CARGO_TARGET_DIR)" $(GO) test ./e2e -run '^TestArenaRunnerJankenRustWASMEvaluationPath$$'

fmt:
	mkdir -p "$(GOPATH)" "$(GOCACHE)" "$(GOMODCACHE)"
	if [ -n "$(GOFILES)" ]; then $(GO_ENV) $(GO) tool goimports -w $(GOFILES); fi

lint: lint-goimports lint-vet lint-noctx lint-staticcheck lint-gosec

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

build-dungeon-go-wasm:
	mkdir -p "$(GOPATH)" "$(GOCACHE)" "$(GOMODCACHE)"
	$(GO_ENV) GOOS=wasip1 GOARCH=wasm $(GO) build -o ./testdata/ai/dungeon/dungeon-go-wasm-ai.wasm ./testdata/ai/dungeon/dungeon-go-wasm-ai

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

run-dungeon-local:
	mkdir -p "$(GOCACHE)" "$(GOMODCACHE)"
	output_dir="$$(mktemp -d /tmp/ai-arena-dungeon-XXXXXX)"; \
	echo "artifact dir: $$output_dir/dungeon-local"; \
	$(GO_ENV) $(GO) run ./cmd/arena-runner \
		--game dungeon \
		--game-version 1.0.0 \
		--ruleset "$(DUNGEON_RULESET)" \
		--match-id dungeon-local \
		--output-dir "$$output_dir" \
		--rng-seed "$(DUNGEON_RNG_SEED)" \
		--player p1=./testdata/ai/dungeon/dungeon-bot-local-seeded \
		--player p2=./testdata/ai/dungeon/dungeon-bot-local-seeded

run-dungeon-local-quiet:
	mkdir -p "$(GOCACHE)" "$(GOMODCACHE)"
	output_dir="$$(mktemp -d /tmp/ai-arena-dungeon-quiet-XXXXXX)"; \
	echo "artifact dir: $$output_dir/dungeon-local"; \
	$(GO_ENV) $(GO) run ./cmd/arena-runner \
		--game dungeon \
		--game-version 1.0.0 \
		--ruleset "$(DUNGEON_RULESET)" \
		--match-id dungeon-local \
		--output-dir "$$output_dir" \
		--log-output none \
		--rng-seed "$(DUNGEON_RNG_SEED)" \
		--player p1=./testdata/ai/dungeon/dungeon-bot-local-seeded \
		--player p2=./testdata/ai/dungeon/dungeon-bot-local-seeded && \
	cat "$$output_dir/dungeon-local/result-summary.json"

run-dungeon-go-wasm: build-dungeon-go-wasm
	mkdir -p "$(GOCACHE)" "$(GOMODCACHE)"
	output_dir="$$(mktemp -d /tmp/ai-arena-dungeon-wasm-XXXXXX)"; \
	echo "artifact dir: $$output_dir/dungeon-go-wasm"; \
	$(GO_ENV) $(GO) run ./cmd/arena-runner \
		--game dungeon \
		--game-version 1.0.0 \
		--ruleset "$(DUNGEON_RULESET)" \
		--match-id dungeon-go-wasm \
		--output-dir "$$output_dir" \
		--rng-seed "$(DUNGEON_RNG_SEED)" \
		--player p1=./testdata/ai/dungeon/dungeon-go-wasm-ai \
		--player p2=./testdata/ai/dungeon/dungeon-bot-local-seeded

inspect-dungeon-map:
	mkdir -p "$(GOCACHE)" "$(GOMODCACHE)"
	$(GO_ENV) $(GO) run ./cmd/dungeon-map-helper \
		--ruleset "$(DUNGEON_RULESET)" \
		--rng-seed "$(DUNGEON_RNG_SEED)"
