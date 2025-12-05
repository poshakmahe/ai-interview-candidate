# SecureVault - Secure Document Storage Application

A full-stack secure document vault application built with Go (backend) and Next.js (frontend). This repository is designed for technical interviews to evaluate candidates' ability to use AI-assisted development tools.

## Technology Stack

### Backend
- **Language**: Go 1.21
- **Framework**: Gin (HTTP router)
- **Database**: PostgreSQL 15
- **Authentication**: JWT tokens
- **Testing**: Go testing package

### Frontend
- **Framework**: Next.js 14 with App Router
- **Language**: TypeScript
- **Styling**: Tailwind CSS
- **State Management**: Zustand
- **Form Handling**: React Hook Form + Zod
- **Testing**: Jest + React Testing Library + Playwright

## Project Structure

```
├── backend/
│   ├── cmd/server/          # Application entry point
│   ├── internal/
│   │   ├── config/          # Configuration management
│   │   ├── database/        # Database connection & migrations
│   │   ├── handlers/        # HTTP handlers
│   │   ├── middleware/      # Auth & CORS middleware
│   │   ├── models/          # Data models & DTOs
│   │   └── services/        # Business logic
│   ├── Dockerfile
│   └── go.mod
├── frontend/
│   ├── src/
│   │   ├── app/             # Next.js pages (App Router)
│   │   ├── components/      # React components
│   │   ├── hooks/           # Custom hooks & state
│   │   ├── services/        # API client
│   │   └── types/           # TypeScript types
│   ├── __tests__/           # Jest tests
│   ├── e2e/                 # Playwright E2E tests
│   ├── Dockerfile
│   └── package.json
├── docker-compose.yml
└── README.md
```

## Quick Start

> **For detailed setup instructions, troubleshooting, and interview preparation, see [SETUP.md](./SETUP.md)**
>
> **For architecture diagrams, development guides, and advanced troubleshooting, see [DEVELOPMENT.md](./DEVELOPMENT.md)**

### Prerequisites
- Docker and Docker Compose
- Node.js 20+ (for local frontend development)
- Go 1.21+ (for local backend development)
- PostgreSQL 15+ (for local development without Docker)

### Using Docker (Recommended)

```bash
# Start all services
docker-compose up -d

# Verify everything is working
./scripts/health-check.sh

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

The application will be available at:
- Frontend: http://localhost:3000
- Backend API: http://localhost:8080
- Database: localhost:5432

### Local Development

#### Backend

```bash
cd backend

# Install dependencies
go mod download

# Set environment variables (or create .env file)
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/docvault?sslmode=disable"
export JWT_SECRET="your-secret-key"
export PORT="8080"

# Run migrations and start server
go run cmd/server/main.go
```

#### Frontend

```bash
cd frontend

# Install dependencies
npm install

# Set environment variables
export NEXT_PUBLIC_API_URL="http://localhost:8080"

# Start development server
npm run dev
```

## API Endpoints

### Authentication
| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/auth/register` | Register new user |
| POST | `/auth/login` | Login user |
| GET | `/auth/me` | Get current user (protected) |

### Documents
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/documents` | List user's documents |
| POST | `/documents` | Upload new document |
| GET | `/documents/:id` | Get document details |
| PATCH | `/documents/:id` | Rename document |
| DELETE | `/documents/:id` | Delete document |
| GET | `/documents/:id/download` | Download document |
| POST | `/documents/:id/share` | Share document |
| GET | `/shared` | List documents shared with user |

## Running Tests

### Backend Tests

```bash
cd backend

# Create test database
createdb docvault_test

# Run tests
go test ./... -v

# Run tests with coverage
go test ./... -cover
```

### Frontend Tests

```bash
cd frontend

# Run Jest tests
npm test

# Run tests with coverage
npm run test:coverage

# Run E2E tests (requires app running)
npm run test:e2e
```

## Features

- **User Authentication**: Register, login, JWT-based session management
- **Document Upload**: Drag-and-drop file upload with progress
- **Document Management**: View, rename, download, delete documents
- **Document Sharing**: Share documents with other users with permission levels (view/edit)
- **AI-Powered Summarization**: Generate concise summaries of documents using Google Gemini API (supports PDF, DOCX, and text files)
- **Responsive Design**: Mobile-friendly UI with Tailwind CSS
- **Security**: File encryption metadata, access control, secure headers

## Environment Variables

### Backend
| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `DATABASE_URL` | PostgreSQL connection string | - |
| `JWT_SECRET` | JWT signing secret | - |
| `UPLOAD_DIR` | File upload directory | `./uploads` |
| `MAX_FILE_SIZE` | Max upload size in bytes | `10485760` (10MB) |
| `ALLOWED_ORIGINS` | CORS allowed origins | `http://localhost:3000` |

### Frontend
| Variable | Description | Default |
|----------|-------------|---------|
| `NEXT_PUBLIC_API_URL` | Backend API URL | `http://localhost:8080` |

## Interview Information

This repository is designed for technical interviews. Candidates should:

1. **Set up the environment** before the interview (see [SETUP.md](./SETUP.md))
2. **Review the architecture** and development guide (see [DEVELOPMENT.md](./DEVELOPMENT.md))
3. **Familiarize themselves** with the codebase structure
4. **Have Claude Code or similar AI assistant** ready to use
5. Be prepared to **demonstrate problem-solving** using AI tools

See [INTERVIEW_GUIDE.md](./INTERVIEW_GUIDE.md) for interviewer instructions and question sets.
