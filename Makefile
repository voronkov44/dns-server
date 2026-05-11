APP_NAME := dns-manager
SERVER_BIN := bin/dns-manager-server
CLI_BIN := bin/dnsctl

ifneq (,$(wildcard .env))
	include .env
endif

DNS_MANAGER_HTTP_ADDR ?= :8080
DNS_MANAGER_RESOLV_CONF_PATH ?= ./resolv.conf
DNS_MANAGER_READ_HEADER_TIMEOUT ?= 5s
DNS_MANAGER_SHUTDOWN_TIMEOUT ?= 10s
DNS_MANAGER_LOG_LEVEL ?= info
DNS_MANAGER_LOG_FILE_PATH ?= logs/dns-server.log
DNS_MANAGER_SERVER_URL ?= http://localhost:8080

export DNS_MANAGER_HTTP_ADDR
export DNS_MANAGER_RESOLV_CONF_PATH
export DNS_MANAGER_READ_HEADER_TIMEOUT
export DNS_MANAGER_SHUTDOWN_TIMEOUT
export DNS_MANAGER_LOG_LEVEL
export DNS_MANAGER_LOG_FILE_PATH
export DNS_MANAGER_SERVER_URL

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  make run-server       Run server"
	@echo "  make run-cli          Show CLI help"
	@echo "  make cli-list         Run dnsctl list"
	@echo "  make cli-add DNS=8.8.8.8"
	@echo "  make cli-delete DNS=8.8.8.8"
	@echo "  make build            Build server and CLI binaries"
	@echo "  make build-server     Build server binary"
	@echo "  make build-cli        Build CLI binary"
	@echo "  make install-cli      Install dnsctl into GOPATH/bin"
	@echo "  make test             Run tests"
	@echo "  make lint             Run golangci-lint"
	@echo "  make check            Run fmt, tidy, lint and tests"
	@echo "  make fmt              Format Go code"
	@echo "  make tidy             Run go mod tidy"
	@echo "  make clean            Remove build artifacts"

.PHONY: run-server
run-server:
	go run ./cmd/server

.PHONY: run-cli
run-cli:
	go run ./cmd/dnsctl --help

.PHONY: cli-list
cli-list:
	go run ./cmd/dnsctl list

.PHONY: cli-add
cli-add:
ifndef DNS
	$(error DNS is required. Usage: make cli-add DNS=8.8.8.8)
endif
	go run ./cmd/dnsctl add $(DNS)

.PHONY: cli-delete
cli-delete:
ifndef DNS
	$(error DNS is required. Usage: make cli-delete DNS=8.8.8.8)
endif
	go run ./cmd/dnsctl delete $(DNS)

.PHONY: build
build: build-server build-cli

.PHONY: build-server
build-server:
	@mkdir -p bin
	go build -o $(SERVER_BIN) ./cmd/server

.PHONY: build-cli
build-cli:
	@mkdir -p bin
	go build -o $(CLI_BIN) ./cmd/dnsctl

.PHONY: install-cli
install-cli:
	go install ./cmd/dnsctl

.PHONY: test
test:
	go test ./...

.PHONY: lint
lint:
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "golangci-lint is not installed"; \
		echo "Install it and run make lint again"; \
		exit 1; \
	fi
	golangci-lint run ./...

.PHONY: check
check: fmt tidy lint test

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: clean
clean:
	rm -rf bin