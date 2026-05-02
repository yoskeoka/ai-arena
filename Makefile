GO ?= go
CACHE_ROOT ?= /tmp/ai-arena-go-quality-gates
GOCACHE = $(CACHE_ROOT)/go-build
GOPATH = $(CACHE_ROOT)/go
GOMODCACHE = $(GOPATH)/pkg/mod
GO_ENV = GOPATH=$(GOPATH) GOMODCACHE=$(GOMODCACHE) GOCACHE=$(GOCACHE)
GOFILES = $(shell git ls-files -- '*.go')

.PHONY: test fmt lint lint-goimports lint-vet lint-noctx lint-staticcheck lint-gosec

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
