# S3 Upload Tool

HTTP API service for uploading files to Amazon S3 with WordPress integration, image resize/optimization, and a React upload UI (Lumen).

## Documentation

- **[Architecture Documentation](./docs/ARCHITECTURE.md)** — architecture, project structure, and design patterns

## Quick Start

### Prerequisites

- Go 1.24+
- Node.js 20+ (for the web UI)
- PostgreSQL (optional, for upload tracking)
- AWS S3 bucket and credentials

### Configuration

Copy `.env.example` to `.env` and configure:

```env
PORT=8080
AWS_REGION=us-east-1
S3_BUCKET_NAME=your-bucket-name
DATABASE_ENABLED=true
DATABASE_URL=postgres://user:pass@localhost/dbname?sslmode=disable
```

### Web UI

```bash
cd web
npm install
npm run build
```

Then start the Go server — the UI is served at `http://localhost:8080` from `web/dist`.

**Dev mode** (Vite hot reload + API proxy):

```bash
# terminal 1
go run cmd/server/main.go

# terminal 2
cd web
npm run dev
```

Open `http://localhost:5173` — `/api` is proxied to `:8080`.

### Run API / server

```bash
# Run server (serves API + UI when web/dist exists)
go run cmd/server/main.go

# Run migrations (if database enabled)
go run cmd/migrate/main.go -up
```

Without a built UI, `/` still returns API endpoint JSON.

## Features

- React upload UI with drag-and-drop, progress, S3 / WP resize modes
- File upload to S3 (≤20MB direct; larger files auto-chunked up to 200MB)
- Chunked upload API: `/api/v1/upload/init` → `/chunk` → `/complete`
- Database upload tracking with transactions
- WordPress image resize and optimization
- Rate limiting and concurrency limiting
- CSRF and CORS protection
- API key authentication
- Health check and metrics endpoints

## License

MIT
