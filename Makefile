SHELL := /bin/bash

SERVICES := bff auth catalog ingestion streaming activity social billing

.PHONY: help
help:
	@printf "Targets:\n"
	@printf "  make tidy        - go mod tidy\n"
	@printf "  make tools       - install goimports + golangci-lint\n"
	@printf "  make fmt         - gofmt (+ goimports if installed)\n"
	@printf "  make lint        - run golangci-lint\n"
	@printf "  make test        - run tests\n"
	@printf "  make build       - build all service binaries into ./bin\n"
	@printf "  make hooks       - enable git hooks (pre-commit fmt+lint)\n"
	@printf "  make migrate-auth-up   - apply auth DB migrations\n"
	@printf "  make migrate-auth-down - rollback auth DB migrations (1 step)\n"
	@printf "  make migrate-catalog-up   - apply catalog DB migrations\n"
	@printf "  make migrate-catalog-down - rollback catalog DB migrations (1 step)\n"
	@printf "  make migrate-activity-up   - apply activity DB migrations\n"
	@printf "  make migrate-activity-down - rollback activity DB migrations (1 step)\n"
	@printf "  make proto       - lint + generate protobuf\n"
	@printf "  make up          - docker compose up --build\n"
	@printf "  make down        - docker compose down -v\n"

.PHONY: tidy
 tidy:
	go mod tidy

.PHONY: test
test:
	go test ./...

.PHONY: tools
GOBIN ?= $(shell go env GOPATH)/bin
GOIMPORTS := $(GOBIN)/goimports
GOLANGCI_LINT := $(GOBIN)/golangci-lint
MIGRATE := $(GOBIN)/migrate
BUF := $(GOBIN)/buf
PROTOC_GEN_GO := $(GOBIN)/protoc-gen-go
PROTOC_GEN_GO_GRPC := $(GOBIN)/protoc-gen-go-grpc

tools:
	@echo "==> installing goimports to $(GOIMPORTS)"
	go install golang.org/x/tools/cmd/goimports@latest
	@echo "==> installing golangci-lint to $(GOLANGCI_LINT)"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "==> installing migrate (postgres driver) to $(MIGRATE)"
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "==> installing buf to $(BUF)"
	go install github.com/bufbuild/buf/cmd/buf@latest
	@echo "==> installing protoc plugins"
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

.PHONY: fmt
fmt:
	@echo "==> gofmt"
	gofmt -w $(shell find . -name '*.go' -not -path './bin/*')
	@echo "==> goimports (optional)"
	@if [ -x "$(GOIMPORTS)" ]; then \
		$(GOIMPORTS) -local github.com/example/anime-platform -w $(shell find . -name '*.go' -not -path './bin/*'); \
	else \
		echo "goimports not installed (recommended): make tools"; \
	fi

.PHONY: lint
lint:
	@if [ -x "$(GOLANGCI_LINT)" ]; then \
		$(GOLANGCI_LINT) run ./...; \
	else \
		echo "golangci-lint not installed: make tools"; \
		exit 1; \
	fi

.PHONY: hooks
hooks:
	@if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then \
		git config core.hooksPath .githooks; \
		echo "Enabled git hooks: core.hooksPath=$$(git config core.hooksPath)"; \
	else \
		echo "Not a git repository. Run: git init && make hooks"; \
	fi

.PHONY: migrate-auth-up
migrate-auth-up:
	@if [ -z "$(DATABASE_URL)" ]; then echo "DATABASE_URL is required"; exit 1; fi
	@if [ ! -x "$(MIGRATE)" ]; then echo "migrate not installed: make tools"; exit 1; fi
	$(MIGRATE) -database "$(DATABASE_URL)" -path services/auth/migrations up

.PHONY: migrate-auth-down
migrate-auth-down:
	@if [ -z "$(DATABASE_URL)" ]; then echo "DATABASE_URL is required"; exit 1; fi
	@if [ ! -x "$(MIGRATE)" ]; then echo "migrate not installed: make tools"; exit 1; fi
	$(MIGRATE) -database "$(DATABASE_URL)" -path services/auth/migrations down 1

.PHONY: migrate-catalog-up
migrate-catalog-up:
	@if [ -z "$(DATABASE_URL)" ]; then echo "DATABASE_URL is required"; exit 1; fi
	@if [ ! -x "$(MIGRATE)" ]; then echo "migrate not installed: make tools"; exit 1; fi
	$(MIGRATE) -database "$(DATABASE_URL)" -path services/catalog/migrations up

.PHONY: migrate-catalog-down
migrate-catalog-down:
	@if [ -z "$(DATABASE_URL)" ]; then echo "DATABASE_URL is required"; exit 1; fi
	@if [ ! -x "$(MIGRATE)" ]; then echo "migrate not installed: make tools"; exit 1; fi
	$(MIGRATE) -database "$(DATABASE_URL)" -path services/catalog/migrations down 1

.PHONY: migrate-activity-up
migrate-activity-up:
	@if [ -z "$(DATABASE_URL)" ]; then echo "DATABASE_URL is required"; exit 1; fi
	@if [ ! -x "$(MIGRATE)" ]; then echo "migrate not installed: make tools"; exit 1; fi
	$(MIGRATE) -database "$(DATABASE_URL)" -path services/activity/migrations up

.PHONY: migrate-activity-down
migrate-activity-down:
	@if [ -z "$(DATABASE_URL)" ]; then echo "DATABASE_URL is required"; exit 1; fi
	@if [ ! -x "$(MIGRATE)" ]; then echo "migrate not installed: make tools"; exit 1; fi
	$(MIGRATE) -database "$(DATABASE_URL)" -path services/activity/migrations down 1

.PHONY: proto
proto:
	@if [ ! -x "$(BUF)" ]; then echo "buf not installed: make tools"; exit 1; fi
	PATH=$(GOBIN):$$PATH $(BUF) lint
	PATH=$(GOBIN):$$PATH $(BUF) generate

.PHONY: build
build:
	mkdir -p bin
	@for s in $(SERVICES); do \
		echo "==> building $$s"; \
		CGO_ENABLED=0 go build -o bin/$$s ./services/$$s/cmd/$$s; \
	done

.PHONY: up
up:
	docker compose up --build

.PHONY: down
down:
	docker compose down -v
