# quanly-phongtro (Clean Architecture)

## Project structure

```text
cmd/api/main.go                             # entrypoint
internal/model/user                         # entities + repository contract
internal/service/auth                       # business rules (register/login)
internal/handler                            # HTTP handlers
internal/router                             # gorilla/mux route setup
internal/config                             # env config
internal/db                                 # postgres connection
internal/repository                         # postgres repository (pq)
internal/security                           # bcrypt + jwt implementation
migrations                                  # SQL schema
```

## Environment

Use `.env.example` as reference:

```env
APP_PORT=8080
POSTGRES_DSN=postgres://postgres:postgres@localhost:5432/quanly_phongtro?sslmode=disable
JWT_SECRET=change-me
JWT_TTL_MINUTES=60
```

## Database Migrations

This project uses `golang-migrate` to manage database schema.

### Execute Migrations

To apply all pending migrations to the database:
```bash
# Load environment variables from .env
export $(grep -v '^#' .env | xargs)

# Run up migrations
migrate -path migrations -database "$POSTGRES_DSN" up
```

### Rollback Migrations

To rollback the last migration:
```bash
migrate -path migrations -database "$POSTGRES_DSN" down 1
```

### Create New Migration

```bash
migrate create -ext sql -dir migrations -seq name_of_migration
```

### Seed Data (Optional)

To insert test data:
```bash
go run ./cmd/seed
```

## Run

```bash
go run ./cmd/api
```

## APIs

- `POST /api/v1/auth/register`
  - body: `{"email":"user@example.com","password":"secret123"}`
- `POST /api/v1/auth/login`
  - body: `{"email":"user@example.com","password":"secret123"}`
- `GET /health`
