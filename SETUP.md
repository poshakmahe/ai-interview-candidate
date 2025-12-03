# Setup Guide for Interview Candidates

This guide will help you set up the SecureVault application before your interview.

## Prerequisites

Before you begin, ensure you have the following installed:

- **Docker Desktop** (recommended) or Docker CLI + Docker Compose
  - Download: https://www.docker.com/products/docker-desktop
  - Minimum version: Docker 20.x, Docker Compose 2.x
- **Git** for cloning the repository
- **Your preferred AI coding assistant** (Claude Code, Cursor, GitHub Copilot, etc.)

### Optional (for local development without Docker):
- **Go 1.21+** for backend
- **Node.js 20+** for frontend
- **PostgreSQL 15+**

---

## Quick Start (Docker - Recommended)

### Step 1: Clone the Repository

```bash
git clone <repository-url>
cd ai-interview-repo
```

### Step 2: Start All Services

```bash
docker-compose up -d
```

This will start:
- PostgreSQL database on port 5432
- Backend API on port 8080
- Frontend web app on port 3000

**First-time setup takes 2-3 minutes** as Docker builds the images.

### Step 3: Verify Setup

Run the health check script:

```bash
./scripts/health-check.sh
```

You should see:
```
✓ Docker is running
✓ Containers running (3/3)
✓ PostgreSQL ready
✓ Backend API responding
✓ Frontend responding
```

### Step 4: Access the Application

Open your browser and navigate to:
- **Frontend**: http://localhost:3000
- **Backend API**: http://localhost:8080/health

You should see the SecureVault landing page.

---

## Common Issues & Solutions

### Issue 1: Port Already in Use

**Error**: `Bind for 0.0.0.0:5432 failed: port is already allocated`

**Solution**: Another service is using the port.

```bash
# Check what's using the port
lsof -i :5432  # or :3000, :8080

# Option 1: Stop the conflicting service
# Option 2: Change ports in docker-compose.yml
```

---

### Issue 2: Docker Daemon Not Running

**Error**: `Cannot connect to the Docker daemon`

**Solution**: Start Docker Desktop

- **Mac**: Open Docker Desktop from Applications
- **Windows**: Start Docker Desktop from Start menu
- **Linux**: `sudo systemctl start docker`

---

### Issue 3: "Failed to connect to database"

**Symptoms**: Backend logs show database connection errors

**Solution**: PostgreSQL container might not be ready yet

```bash
# Wait 10 seconds and check again
docker-compose logs postgres

# If it's still failing, restart
docker-compose restart postgres backend
```

---

### Issue 4: Frontend Shows "Network Error"

**Symptoms**: Can access http://localhost:3000 but API calls fail

**Solution**: Backend might not be running

```bash
# Check backend status
curl http://localhost:8080/health

# Check logs
docker-compose logs backend

# If JWT_SECRET error, backend won't start (this is expected - it's configured in docker-compose.yml)
```

---

### Issue 5: Stale Data or Weird Errors

**Solution**: Full reset

```bash
# Stop and remove all containers and volumes
docker-compose down -v

# Start fresh
docker-compose up -d

# Wait 30 seconds, then verify
./scripts/health-check.sh
```

---

## Local Development Setup (Without Docker)

If you prefer to run services locally:

### Backend Setup

```bash
cd backend

# Install dependencies
go mod download

# Set environment variables
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/docvault?sslmode=disable"
export JWT_SECRET="dev-secret-for-local-testing"
export PORT="8080"

# Ensure PostgreSQL is running and create database
createdb docvault

# Run the server (migrations run automatically)
go run cmd/server/main.go
```

### Frontend Setup

```bash
cd frontend

# Install dependencies
npm install

# Set environment variables
export NEXT_PUBLIC_API_URL="http://localhost:8080"

# Start development server
npm run dev
```

---

## Verifying Your Setup

### 1. Test User Registration

```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123",
    "name": "Test User"
  }'
```

You should receive a JSON response with a token and user object.

### 2. Test Frontend

1. Go to http://localhost:3000
2. Click "Sign Up"
3. Create an account
4. You should be redirected to the dashboard

### 3. Test File Upload

1. Click "Upload Document" on the dashboard
2. Select a file (< 10MB)
3. Upload should succeed
4. Document should appear in the list

---

## Interview Day Checklist

Before your interview, ensure:

- [ ] All services are running (`./scripts/health-check.sh` passes)
- [ ] You can access http://localhost:3000 and see the app
- [ ] You have tested registration/login
- [ ] You have uploaded at least one test document
- [ ] Your AI coding assistant is configured and ready
- [ ] You've familiarized yourself with the codebase structure
- [ ] You know how to view logs: `docker-compose logs -f`

---

## Useful Commands

```bash
# Start services
docker-compose up -d

# View logs (all services)
docker-compose logs -f

# View logs (specific service)
docker-compose logs -f backend
docker-compose logs -f frontend
docker-compose logs -f postgres

# Stop services
docker-compose down

# Restart a service
docker-compose restart backend

# Full reset (delete all data)
docker-compose down -v
docker-compose up -d

# Check container status
docker-compose ps

# Execute command in container
docker-compose exec backend sh
docker-compose exec postgres psql -U postgres -d docvault
```

---

## Project Structure Overview

```
├── backend/                 # Go API
│   ├── cmd/server/         # Main entry point
│   ├── internal/
│   │   ├── handlers/       # HTTP handlers
│   │   ├── services/       # Business logic
│   │   ├── database/       # DB connection
│   │   └── middleware/     # Auth, CORS
│   └── pkg/utils/          # Utilities
│
├── frontend/               # Next.js app
│   ├── src/
│   │   ├── app/           # Pages (Next.js 14 App Router)
│   │   ├── components/    # React components
│   │   ├── hooks/         # Custom hooks (state)
│   │   ├── services/      # API client
│   │   └── types/         # TypeScript types
│   └── __tests__/         # Tests
│
├── scripts/               # Utility scripts
└── docker-compose.yml     # Service orchestration
```

---

## Getting Help

If you encounter issues not covered here:

1. **Check logs**: `docker-compose logs -f`
2. **Restart services**: `docker-compose restart`
3. **Full reset**: `docker-compose down -v && docker-compose up -d`
4. **Contact the interviewer** if problems persist

---

## What to Explore Before Interview

Recommended areas to familiarize yourself with:

1. **API Endpoints**: Check `backend/cmd/server/main.go` for routes
2. **Frontend Pages**: Explore `frontend/src/app/` directory
3. **Data Models**: See `backend/internal/models/` and `frontend/src/types/`
4. **Components**: Browse `frontend/src/components/`
5. **Tests**: Look at `backend/internal/handlers/*_test.go` and `frontend/__tests__/`

---

## Performance Notes

- **First startup**: 2-3 minutes (building Docker images)
- **Subsequent startups**: 10-30 seconds
- **Frontend hot reload**: Works in development mode
- **Backend changes**: Require rebuild (`docker-compose up -d --build backend`)

---

## Security Notes

- The default `JWT_SECRET` is set in docker-compose.yml for convenience
- This is a development/interview environment - **not production-ready**
- File uploads are stored in Docker volumes
- Database credentials are default (postgres/postgres)

---

## Troubleshooting Decision Tree

```
Can't access localhost:3000?
  ├─ Is Docker running? → Start Docker Desktop
  ├─ Are containers running? → docker-compose up -d
  └─ Check logs → docker-compose logs frontend

Backend errors?
  ├─ Database connection? → docker-compose restart postgres backend
  ├─ JWT_SECRET error? → Check docker-compose.yml (should be set)
  └─ Check logs → docker-compose logs backend

Database errors?
  ├─ Container running? → docker-compose ps
  ├─ Migration failed? → docker-compose down -v && docker-compose up -d
  └─ Check logs → docker-compose logs postgres
```

---

**Good luck with your interview! 🚀**
