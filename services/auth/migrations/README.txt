Auth service DB migrations.

We use golang-migrate.

Apply:
  make migrate-auth-up DATABASE_URL=postgres://...
Rollback:
  make migrate-auth-down DATABASE_URL=postgres://...

Note: DATABASE_URL is required.
