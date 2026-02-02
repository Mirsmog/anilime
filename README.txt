anime-platform (skeleton)

Local dev:
  make tidy
  make test
  make build
  make up

Formatting & linting:
  make tools   # installs goimports + golangci-lint
  make fmt     # gofmt (+ goimports if installed)
  make lint    # golangci-lint (requires tools)

Git hooks:
  make hooks   # enables .githooks/pre-commit (runs fmt+lint before commit)

Install tools (one-time):
  make tools
  # or manually:
  # go install golang.org/x/tools/cmd/goimports@latest
  # go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

Services:
  bff      :8080
  auth     :8081
  catalog  :8082
  ingestion:8083
  streaming:8084
  activity :8085
  social   :8086
  billing  :8087

Infra:
  postgres :5432 (app/app)
  redis    :6379
  nats     :4222 (JetStream enabled), monitoring :8222
  meili    :7700 (master key: dev-master-key)

Health:
  GET /healthz
  GET /readyz

Note: This repo uses REST externally. gRPC/proto for internal contracts is planned.
