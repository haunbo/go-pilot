GO ?= go

.PHONY: help test test-race vet build build-nocgo fmt tidy

help:
	@echo "make test       - run unit tests"
	@echo "make test-race  - run unit tests with the race detector"
	@echo "make vet        - run go vet"
	@echo "make build      - build all packages"
	@echo "make build-nocgo- build all packages without CGO"
	@echo "make fmt        - format all Go files"
	@echo "make tidy       - tidy go.mod"

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

build:
	$(GO) build ./...

build-nocgo:
	CGO_ENABLED=0 $(GO) build ./...

fmt:
	gofmt -w $$(find pkg -name '*.go')

tidy:
	$(GO) mod tidy
