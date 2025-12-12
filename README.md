# S3 Upload Tool

HTTP API service for uploading files to Amazon S3 with WordPress integration, image resize/optimization, and comprehensive security features.

## 📖 Documentation

- **[Architecture Documentation](./docs/ARCHITECTURE.md)** - Chi tiết về kiến trúc, cấu trúc project, và design patterns

## 🚀 Quick Start

### Prerequisites

- Go 1.24+
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

### Run

```bash
# Run server
go run cmd/server/main.go

# Run migrations (if database enabled)
go run cmd/migrate/main.go -up
```

## 📚 Features

- ✅ File upload to S3
- ✅ Database upload tracking with transactions
- ✅ WordPress image resize and optimization
- ✅ Rate limiting and concurrency limiting
- ✅ CSRF and CORS protection
- ✅ API key authentication
- ✅ Health check and metrics endpoints

## 📝 License

MIT 
