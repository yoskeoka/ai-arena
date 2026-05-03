GO ?= go
CACHE_ROOT ?= /tmp/ai-arena-go-quality-gates
GOCACHE = $(CACHE_ROOT)/go-build
GOPATH = $(CACHE_ROOT)/go
GOMODCACHE = $(GOPATH)/pkg/mod
GO_ENV = GOPATH=$(GOPATH) GOMODCACHE=$(GOMODCACHE) GOCACHE=$(GOCACHE)
GOFILES = $(shell git ls-files -- '*.go')

.PHONY: test fmt lint lint-goimports lint-vet lint-noctx lint-staticcheck lint-gosec run-echo-simultaneous run-echo-sequential

test:
	mkdir -p "$(GOCACHE)" "$(GOMODCACHE)"
	$(GO_ENV) $(GO) test ./...

fmt:
	mkdir -p "$(GOCACHE)" "$(GOMODCACHE)"
	if [ -n "$(GOFILES)" ]; then $(GO_ENV) $(GO) tool goimports -w $(GOFILES); fi

lint: lint-goimports lint-vet lint-noctx lint-staticcheck lint-gosec

lint-goimports:
	mkdir -p "$(GOCACHE)" "$(GOMODCACHE)"
	if [ -n "$(GOFILES)" ]; then \
		output="$$( $(GO_ENV) $(GO) tool goimports -l $(GOFILES) )"; \
		if [ -n "$$output" ]; then \
			printf '%s\n' "$$output"; \
			exit 1; \
		fi; \
	fi

lint-vet:
	mkdir -p "$(GOCACHE)" "$(GOMODCACHE)"
	$(GO_ENV) $(GO) vet ./...

lint-noctx:
	mkdir -p "$(GOCACHE)" "$(GOMODCACHE)"
	noctx_bin="$$( $(GO_ENV) $(GO) tool -n noctx )"; \
	$(GO_ENV) $(GO) vet -vettool="$$noctx_bin" ./...

lint-staticcheck:
	mkdir -p "$(GOCACHE)" "$(GOMODCACHE)"
	$(GO_ENV) $(GO) tool staticcheck ./...

lint-gosec:
	mkdir -p "$(GOCACHE)" "$(GOMODCACHE)"
	$(GO_ENV) $(GO) tool gosec -exclude-dir=.cache ./...

run-echo-simultaneous:
	mkdir -p "$(GOCACHE)" "$(GOMODCACHE)"
	$(GO_ENV) $(GO) run ./cmd/arena-runner \
		--game echo-count \
		--game-version 2.0.0 \
		--ruleset phase2-simultaneous-3turn \
		--match-id sim-happy \
		--player p1=./testdata/ai/echo/echo-ai \
		--player p2=./testdata/ai/echo/echo-ai

run-echo-sequential:
	mkdir -p "$(GOCACHE)" "$(GOMODCACHE)"
	$(GO_ENV) $(GO) run ./cmd/arena-runner \
		--game echo-count \
		--game-version 2.0.0 \
		--ruleset phase2-sequential-3turn \
		--match-id seq-happy \
		--player p1=./testdata/ai/echo/echo-ai-sequential \
		--player p2=./testdata/ai/echo/echo-ai-sequential
