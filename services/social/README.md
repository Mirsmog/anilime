# Social Service

The social service provides ratings and **comments** (Reddit-like threaded discussions) for anime.

## Environment Variables

| Variable        | Required | Description                                                                 |
|-----------------|----------|-----------------------------------------------------------------------------|
| `SERVICE_NAME`  | yes      | Must be set (e.g. `social`).                                                |
| `DATABASE_URL`  | prod     | Postgres DSN, e.g. `postgres://user:pass@localhost:5432/social?sslmode=disable`. |
| `JWT_SECRET`    | yes      | Shared HMAC secret for verifying JWT tokens.                                |
| `APP_ENV`       | no       | Set to `production` to enforce Postgres (exits if unavailable).             |
| `HTTP_ADDR`     | no       | Listen address (default `:8080`).                                           |
| `LOG_LEVEL`     | no       | Zap log level (default `info`).                                             |

## Development Fallback (InMemory)

If `DATABASE_URL` is not set and `APP_ENV` is **not** `production`, both the ratings and comments stores fall back to an **in-memory implementation**. This allows running the service locally without Postgres:

```bash
export SERVICE_NAME=social
export JWT_SECRET=dev-secret
go run ./services/social/cmd/social
```

> **Warning:** in-memory stores lose all data on restart and are not suitable for production.

## Migrations

Apply with the `migrate` CLI tool (install via `make tools`):

```bash
# Apply all migrations
DATABASE_URL=postgres://... make migrate-social-up

# Rollback one step
DATABASE_URL=postgres://... make migrate-social-down
```

Migrations live in `services/social/migrations/`.

## Comments API

### Create Comment
```
POST /v1/comments/{anime_id}
Authorization: Bearer <jwt>
{"body": "Great episode!", "parent_id": "optional-uuid"}
→ 201 Created
```

### List Thread (paginated tree)
```
GET /v1/comments/{anime_id}?sort=new|top&limit=50&cursor=...
→ 200 OK  {"comments": [...], "next_cursor": "..."}
```
No authentication required.

### Vote
```
POST /v1/comments/{comment_id}/vote
Authorization: Bearer <jwt>
{"vote": 1}   // or -1
→ 204 No Content
```

### Update (author only)
```
PUT /v1/comments/{comment_id}
Authorization: Bearer <jwt>
{"body": "Edited text"}
→ 204 No Content
```

### Delete (author only, soft)
```
DELETE /v1/comments/{comment_id}
Authorization: Bearer <jwt>
→ 204 No Content
```
Body is replaced with `[deleted]` and `deleted_at` is set.
