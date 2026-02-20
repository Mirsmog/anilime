# Anilime

[![CI](https://github.com/mirsmog/anilime/actions/workflows/ci.yml/badge.svg)](https://github.com/mirsmog/anilime/actions/workflows/ci.yml)

Anilime is a Go-based anime streaming platform backend. The repository is a
production-oriented monorepo with a REST BFF, internal gRPC services, PostgreSQL
persistence, NATS JetStream event processing, Meilisearch indexing, signed HLS
proxying, billing webhooks, and analytics publishing.

The public API is exposed through the BFF service. Internal service contracts are
defined with protobuf and generated into `gen/*`.

## Features

- REST gateway for authentication, catalog browsing, search, playback, comments,
  ratings, user profile, and watch progress.
- Internal gRPC services for auth, catalog, activity, search, social, and
  streaming resolution.
- Catalog ingestion from Jikan and streaming metadata providers with scheduled
  refresh jobs and on-demand sync triggers.
- Meilisearch-backed anime search with local cache and Jikan fallback for empty
  search results.
- Signed HLS proxy for playlist and segment access through the platform domain.
- PostgreSQL-backed auth, catalog, activity, billing, and social data.
- NATS JetStream event bus for ingestion, activity, social, billing, analytics,
  and search indexing workflows.
- Stripe webhook handling with idempotency support.
- PostHog analytics consumer for product events.
- CI pipeline for linting, tests, and build validation.

## Architecture

```text
Client
  |
  v
BFF REST API (:8080)
  |
  +-- auth gRPC (:9091)              PostgreSQL
  +-- catalog gRPC (:9092)           PostgreSQL
  +-- activity gRPC (:9093)          PostgreSQL
  +-- search gRPC (:9094)            Meilisearch
  +-- streaming-resolver gRPC (:9095) Redis / external providers
  +-- social gRPC (:9096)            PostgreSQL
  |
  +-- NATS JetStream events
  +-- HLS proxy (:8084)

Workers and integrations:
  ingestion, billing, analytics, catalog outbox, activity worker, social worker
```

Repository layout:

```text
services/          Service implementations and Dockerfiles
internal/platform/ Shared infrastructure packages
proto/             Protobuf service contracts
gen/               Generated Go protobuf clients and servers
docs/openapi.yaml  Public BFF OpenAPI specification
docs/adr/          Architecture decision records
deployments/       Local infrastructure bootstrap files
```

## Services

| Service | Purpose | Default port |
| --- | --- | --- |
| `bff` | Public REST API and request aggregation | `8080` |
| `auth` | Users, credentials, JWT access tokens, refresh sessions | `9091` |
| `catalog` | Anime, episodes, metadata, catalog outbox | `9092` |
| `activity` | Watch progress and continue-watching data | `9093` |
| `search` | Search indexing and query service | `9094` |
| `streaming-resolver` | Playback source resolution | `9095` |
| `social` | Comments, votes, and ratings | `8086`, `9096` |
| `billing` | Stripe webhook ingestion and subscription state | `8087` |
| `ingestion` | External catalog and provider synchronization | `8083` |
| `hls-proxy` | Signed HLS playlist and segment proxy | `8084` |
| `analytics` | NATS to PostHog event consumer | none |

## Requirements

- Go 1.24 or newer
- Docker and Docker Compose
- `make`
- Optional local tools installed by `make tools`: `goimports`, `golangci-lint`,
  `migrate`, `buf`, `protoc-gen-go`, and `protoc-gen-go-grpc`

## Quick Start

Create an environment file:

```bash
cp .env.example .env
```

Fill in strong local secrets in `.env`, then start the full stack:

```bash
make up
```

The BFF API is available at:

```text
http://localhost:8080
```

Useful local endpoints:

```text
GET http://localhost:8080/healthz
GET http://localhost:8080/readyz
GET http://localhost:8080/v1/search?q=naruto
GET http://localhost:8080/v1/anime
```

Stop and remove local containers and volumes:

```bash
make down
```

## Development

Install development tools:

```bash
make tools
```

Run formatting, linting, tests, and builds:

```bash
make fmt
make lint
make test
make build
```

Generate protobuf code:

```bash
make proto
```

Enable the repository pre-commit hook:

```bash
make hooks
```

## Configuration

The local stack reads configuration from `.env`. Start from `.env.example` and
provide real values before running Docker Compose.

Common variables:

| Variable | Description |
| --- | --- |
| `JWT_SECRET` | Secret used for JWT signing. Use a strong value in every environment. |
| `POSTGRES_PASSWORD` | Password for the local PostgreSQL instance. |
| `MEILI_MASTER_KEY` | Meilisearch master key. |
| `HLS_SIGNING_SECRET` | HMAC secret for signed HLS URLs. |
| `STRIPE_SECRET_KEY` | Stripe API key for billing flows. |
| `STRIPE_WEBHOOK_SECRET` | Stripe webhook signing secret. |
| `POSTHOG_API_KEY` | PostHog project API key for analytics publishing. |
| `POSTHOG_HOST` | PostHog host. Defaults to PostHog Cloud. |

Service-level configuration is defined in `docker-compose.yml` and each service
configuration package.

## Database Migrations

Migration targets require `DATABASE_URL` and the `migrate` CLI from `make tools`.

```bash
make migrate-auth-up DATABASE_URL=postgres://app:app@localhost:5432/auth?sslmode=disable
make migrate-catalog-up DATABASE_URL=postgres://app:app@localhost:5432/catalog?sslmode=disable
make migrate-activity-up DATABASE_URL=postgres://app:app@localhost:5432/activity?sslmode=disable
make migrate-billing-up DATABASE_URL=postgres://app:app@localhost:5432/billing?sslmode=disable
make migrate-social-up DATABASE_URL=postgres://app:app@localhost:5432/social?sslmode=disable
```

Rollback targets are also available:

```bash
make migrate-auth-down
make migrate-catalog-down
make migrate-activity-down
make migrate-billing-down
make migrate-social-down
```

## API Documentation

- Public REST API: [docs/openapi.yaml](docs/openapi.yaml)
- Internal contracts: [proto/](proto/)
- Generated Go code: [gen/](gen/)
- Architecture decisions: [docs/adr/](docs/adr/)

Authentication-protected REST endpoints expect a Bearer JWT in the
`Authorization` header.

## Production Notes

Set `APP_ENV=production` for production deployments. Services enforce stricter
startup checks in production:

- Required PostgreSQL connections must be configured and reachable.
- In-memory fallbacks for persistent data are disabled.
- Billing idempotency must use Redis or PostgreSQL.
- NATS connection failures are treated as fatal for services that require the
  event bus.
- Secrets must be provided through the deployment environment, not committed to
  the repository.

Run the CI-equivalent checks before releasing:

```bash
make fmt
make lint
make test
make build
```

## Contributing

1. Keep domain logic inside the owning service.
2. Use `internal/platform/*` only for shared infrastructure concerns.
3. Update protobuf definitions and generated code together.
4. Add or update tests for behavior changes.
5. Run formatting, linting, tests, and builds before opening a pull request.

## License

This repository does not currently include a license file. Add a project license
before accepting external contributions or publishing redistribution terms.
