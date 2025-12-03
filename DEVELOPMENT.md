# Development Guide

Technical documentation for interview candidates working with the SecureVault codebase.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                    Frontend (Next.js 14)                        │
│                      localhost:3000                             │
│  App Router │ Zustand State │ Axios API Client │ Tailwind CSS  │
└──────────────────────────┬──────────────────────────────────────┘
                           │ REST API (JSON) + JWT Bearer Token
┌──────────────────────────▼──────────────────────────────────────┐
│                     Backend (Go/Gin)                            │
│                      localhost:8080                             │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ Middleware: CORS → JWT Auth → Handlers → Services → DB     │ │
│  └────────────────────────────────────────────────────────────┘ │
│  Routes:                                                        │
│  - POST /auth/register, /auth/login, GET /auth/me              │
│  - GET/POST /documents, GET/PATCH/DELETE /documents/:id        │
│  - GET /documents/:id/download, POST /documents/:id/share      │
│  - GET /shared                                                  │
└──────────────────────────┬──────────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│                  PostgreSQL 15 (localhost:5432)                 │
│  Tables: users, documents, document_shares (all UUID PKs)      │
└─────────────────────────────────────────────────────────────────┘
```

### Technology Stack

| Layer | Technologies |
|-------|-------------|
| Frontend | Next.js 14, TypeScript, Tailwind CSS, Zustand, React Hook Form + Zod |
| Backend | Go 1.21, Gin, `database/sql` + `lib/pq`, `golang-jwt/jwt/v5` |
| Database | PostgreSQL 15 |
| Infrastructure | Docker & Docker Compose |

---

## Project Structure

```
backend/
├── cmd/server/main.go           # Entry point, routes
├── internal/
│   ├── handlers/                # HTTP handlers (auth, document)
│   ├── services/                # Business logic (user, document)
│   ├── models/models.go         # All models & DTOs
│   ├── middleware/              # Auth, CORS
│   ├── database/database.go     # DB connection & migrations
│   └── config/config.go         # Environment config
├── pkg/utils/                   # Validation utilities
└── go.mod

frontend/
├── src/
│   ├── app/                     # Pages (login, register, dashboard, shared)
│   ├── components/              # UI components, documents/, layout/
│   ├── services/api.ts          # API client (singleton)
│   ├── hooks/                   # useAuth.ts, useDocuments.ts
│   └── types/index.ts           # All TypeScript interfaces
└── package.json
```

---

## Data Models

### Backend Models (`backend/internal/models/models.go`)

```go
type User struct {
    ID        uuid.UUID `json:"id"`
    Email     string    `json:"email"`
    Password  string    `json:"-"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type Document struct {
    ID             uuid.UUID  `json:"id"`
    OwnerID        uuid.UUID  `json:"owner_id"`
    Name           string     `json:"name"`
    OriginalName   string     `json:"original_name"`
    Size           int64      `json:"size"`
    MimeType       string     `json:"mime_type"`
    IsEncrypted    bool       `json:"is_encrypted"`
    CreatedAt      time.Time  `json:"created_at"`
    DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}

type DocumentShare struct {
    ID           uuid.UUID  `json:"id"`
    DocumentID   uuid.UUID  `json:"document_id"`
    SharedByID   uuid.UUID  `json:"shared_by_id"`
    SharedWithID uuid.UUID  `json:"shared_with_id"`
    Permission   string     `json:"permission"` // "view" or "edit"
    ExpiresAt    *time.Time `json:"expires_at,omitempty"`
}
```

### Database Schema

```sql
-- users: id (UUID PK), email (UNIQUE), password, name, created_at, updated_at
-- documents: id (UUID PK), owner_id (FK), name, original_name, size, mime_type,
--            encryption_key, encryption_algo, file_path, is_encrypted,
--            created_at, updated_at, deleted_at
-- document_shares: id (UUID PK), document_id (FK), shared_by_id (FK),
--                  shared_with_id (FK), permission, expires_at, created_at
```

---

## Running Tests

### Backend

```bash
cd backend
go test ./... -v              # All tests
go test ./... -cover          # With coverage
go test ./internal/handlers -run TestUploadDocument -v  # Specific test
```

### Frontend

```bash
cd frontend
npm test                      # Jest unit tests
npm run test:coverage         # With coverage
npm run test:e2e              # Playwright E2E (requires app running)
npm run test:e2e:ui           # E2E with UI
```

---

## Common Commands

```bash
# Start services
docker-compose up -d

# View logs
docker-compose logs -f backend

# Restart after backend changes
docker-compose restart backend

# Reset database
docker-compose down -v && docker-compose up -d

# Access database
docker-compose exec postgres psql -U postgres -d docvault
```

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Port 5432 in use | `brew services stop postgresql` or change port in docker-compose.yml |
| Node version error | `nvm install 20 && nvm use 20` |
| Go module issues | `go clean -modcache && go mod tidy` |
| Frontend build fails | `rm -rf node_modules .next && npm install` |
| CORS errors | Verify `ALLOWED_ORIGINS` in backend and `NEXT_PUBLIC_API_URL` in frontend |
| 401 errors | Clear localStorage and re-login |
| Docker space issues | `docker system prune -a --volumes` |

### Debug Logs

```bash
# Backend: set GIN_MODE=debug or check docker-compose logs
# Frontend: API client logs requests/responses to console on error
```
