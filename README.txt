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

Environment Variables:
  Common:
    SERVICE_NAME      - service identifier (required)
    HTTP_ADDR         - HTTP listen address (default :8080)
    LOG_LEVEL         - log level: debug|info|warn|error (default info)
    DATABASE_URL      - Postgres connection string (required for persistence)
    APP_ENV           - set to "production" to enable production checks (see below)

  NATS (services using event bus: billing, catalog, ingestion, search, bff):
    NATS_URL              - NATS server URL (default nats://nats:4222)
    NATS_MAX_RECONNECTS   - max reconnect attempts before fail-fast (default 5)
    NATS_RECONNECT_WAIT   - wait between reconnect attempts, Go duration (default 2s)

  Billing (services/billing):
    STRIPE_WEBHOOK_SECRET - Stripe webhook signing secret (REQUIRED, fail-fast)
    REDIS_DSN             - Redis URL for idempotency SETNX (e.g. redis://redis:6379/1)
                            Falls back to Postgres INSERT ON CONFLICT if not set.
                            In-memory fallback is FORBIDDEN in production (APP_ENV=production).
    NATS_URL              - NATS server URL for event publishing (default nats://nats:4222)
                            In production, NATS failure causes exit(1) (fail-fast).
    IDEMPOTENCY_TTL_HOURS - How long processed event IDs are retained (default 24)
    DATABASE_URL          - Postgres DSN. Required when APP_ENV=production.

  Social (services/social):
    DATABASE_URL      - Postgres connection string for ratings/comments persistence.
                        If not set, uses in-memory store (development only).
                        Required when APP_ENV=production; ping is verified at startup.

Production Checks (APP_ENV=production):
  When APP_ENV=production, services enforce stricter startup requirements:
  - DATABASE_URL must be set and Postgres must respond to Ping
    (billing, social; auth/catalog/activity already require it unconditionally)
  - Idempotency store must use Redis (REDIS_DSN) or Postgres (DATABASE_URL);
    in-memory fallback is not allowed
  - NATS connection failures cause immediate exit (fail-fast)
  - InMemory stores (ratings, comments) are not permitted

Migrations:
  make migrate-billing-up   DATABASE_URL=postgres://app:app@localhost:5432/billing?sslmode=disable
  make migrate-billing-down DATABASE_URL=postgres://app:app@localhost:5432/billing?sslmode=disable
  make migrate-social-up    DATABASE_URL=postgres://app:app@localhost:5432/social?sslmode=disable
  make migrate-social-down  DATABASE_URL=postgres://app:app@localhost:5432/social?sslmode=disable

Stub / Fallback behavior:
  - Billing: if REDIS_DSN is empty, idempotency uses Postgres. If DATABASE_URL is also
    empty, falls back to in-memory (dev/test only; forbidden in production).
    NATS publisher uses stub mode without NATS (dev only; forbidden in production).
  - Social: if DATABASE_URL is empty, uses in-memory rating/comment store
    (dev/test only; forbidden in production).
  - All NATS-connected services use configurable reconnect (NATS_MAX_RECONNECTS,
    NATS_RECONNECT_WAIT) and fail-fast if connection cannot be established.
