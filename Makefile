GO ?= go
CACHE_ROOT ?= /tmp/ai-arena-go-quality-gates
GOPATH = $(CACHE_ROOT)/go
GOMODCACHE = $(GOPATH)/pkg/mod
GOCACHE = $(CACHE_ROOT)/go-build
GO_ENV = GOPATH=$(GOPATH) GOMODCACHE=$(GOMODCACHE) GOCACHE=$(GOCACHE)
GOFILES = $(shell git ls-files -- '*.go')

.PHONY: test fmt lint lint-goimports lint-vet lint-noctx lint-staticcheck lint-gosec run-echo-simultaneous run-echo-sequential

test:
	mkdir -p "$(GOPATH)" "$(GOCACHE)" "$(GOMODCACHE)"
	$(GO_ENV) $(GO) test ./...

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
