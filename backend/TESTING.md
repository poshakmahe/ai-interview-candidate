# Backend Testing Guide

## Prerequisites

- Go 1.21+
- Docker (for integration tests)

## Quick Start

```bash
# Run unit tests only (no database required)
CGO_ENABLED=0 go test ./internal/config ./internal/middleware ./internal/services ./pkg/utils -v

# Run ALL tests (requires database on port 5433 to avoid conflict with dev DB)
docker run -d --name docvault-test-db -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=docvault_test -p 5433:5432 postgres:15
sleep 3
CGO_ENABLED=0 go test ./... -v
docker stop docvault-test-db && docker rm docvault-test-db
```

## Test Files

| File | Package | Type | DB Required |
|------|---------|------|-------------|
| `validation_test.go` | `pkg/utils` | Unit | No |
| `config_test.go` | `internal/config` | Unit | No |
| `auth_test.go` | `internal/middleware` | Unit | No |
| `cors_test.go` | `internal/middleware` | Unit | No |
| `user_service_test.go` | `internal/services` | Unit (mocked) | No |
| `document_service_test.go` | `internal/services` | Unit (mocked) | No |
| `auth_handler_test.go` | `internal/handlers` | Integration | **Yes** |
| `document_handler_test.go` | `internal/handlers` | Integration | **Yes** |

## Running Tests

### Unit Tests (No Database)

```bash
# All unit tests
CGO_ENABLED=0 go test ./internal/config ./internal/middleware ./internal/services ./pkg/utils -v

# Specific package
CGO_ENABLED=0 go test ./pkg/utils -v
CGO_ENABLED=0 go test ./internal/middleware -v
```

### Integration Tests (Database Required)

```bash
# 1. Start PostgreSQL on port 5433 (avoids conflict with dev DB on 5432)
docker run -d \
  --name docvault-test-db \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=docvault_test \
  -p 5433:5432 \
  postgres:15

# 2. Wait for database to be ready
sleep 3

# 3. Run handler tests
CGO_ENABLED=0 go test ./internal/handlers -v

# 4. Cleanup
docker stop docvault-test-db && docker rm docvault-test-db
```

### All Tests

```bash
# Start database, run all tests, cleanup
docker run -d --name docvault-test-db -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=docvault_test -p 5433:5432 postgres:15 && \
sleep 3 && \
CGO_ENABLED=0 go test ./... -v ; \
docker stop docvault-test-db && docker rm docvault-test-db
```

## Coverage Reports

```bash
# Generate coverage
CGO_ENABLED=0 go test ./... -coverprofile=coverage.out

# View in terminal
go tool cover -func=coverage.out

# View in browser
go tool cover -html=coverage.out
```

## Current Coverage

| Package | Coverage |
|---------|----------|
| `internal/config` | 100% |
| `internal/middleware` | 98.5% |
| `internal/services` | 74.5% |
| `pkg/utils` | 100% |

## Troubleshooting

### `dyld: missing LC_UUID` error (macOS Apple Silicon)
Always use `CGO_ENABLED=0` prefix:
```bash
CGO_ENABLED=0 go test ./... -v
```

### Database connection refused
Ensure PostgreSQL container is running:
```bash
docker ps | grep docvault-test-db
```

### Port 5433 already in use
Check for existing test container or use different port:
```bash
docker ps | grep docvault-test-db
docker stop docvault-test-db && docker rm docvault-test-db
# Or use custom port via environment variable:
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5434/docvault_test?sslmode=disable" go test ./internal/handlers -v
```

## Test Database Details

- **Host:** localhost
- **Port:** 5433 (default, avoids conflict with dev DB on 5432)
- **Database:** docvault_test
- **User:** postgres
- **Password:** postgres

Override with `TEST_DATABASE_URL` environment variable if needed.

Tests automatically run migrations and clean up data between runs.
