# Tài liệu Kiến trúc & Cấu trúc Dự án S3 Upload Tool

## 📋 Mục lục

1. [Tổng quan Kiến trúc](#1-tổng-quan-kiến-trúc)
2. [Cấu trúc Folder](#2-cấu-trúc-folder)
3. [Công nghệ và Package](#3-công-nghệ-và-package)
4. [Vòng đời Hoạt động](#4-vòng-đời-hoạt-động)
5. [Luồng chảy Xử lý từ URL](#5-luồng-chảy-xử-lý-từ-url)
6. [Module / Feature Map](#6-module--feature-map)
7. [Data Flow (Backend)](#7-data-flow-backend)
8. [Entity & Database Overview](#8-entity--database-overview)
9. [Summary](#9-summary)

---

## 1. Tổng quan Kiến trúc

### 1.1. Mô tả High-level

**S3 Upload Tool** là một **HTTP API service** được viết bằng **Golang**, được thiết kế để:

- Upload file lên **Amazon S3** với các tính năng bảo mật và tối ưu
- Hỗ trợ **WordPress integration** với image resize và optimization
- Quản lý upload records trong database (PostgreSQL) với transaction support
- Cung cấp metrics và monitoring endpoints
- Áp dụng rate limiting, concurrency limiting, CORS, CSRF protection

### 1.2. Kiến trúc Pattern

Dự án áp dụng **Clean Architecture** với các đặc điểm:

#### **Layered Architecture** (3 lớp chính):

```
┌─────────────────────────────────────────┐
│         Presentation Layer              │
│  (Handlers - HTTP Request/Response)     │
└─────────────────────────────────────────┘
                  ↓
┌─────────────────────────────────────────┐
│          Business Logic Layer           │
│  (Services - Business Rules)            │
└─────────────────────────────────────────┘
                  ↓
┌─────────────────────────────────────────┐
│          Data Access Layer              │
│  (Repository - Database/S3)             │
└─────────────────────────────────────────┘
```

#### **Dependency Injection (DI) Container Pattern**:
- `internal/container/container.go` quản lý toàn bộ dependencies
- Factory Pattern cho Repository và Service
- Constructor Injection

#### **Repository Pattern**:
- Tách biệt data access logic khỏi business logic
- Interface-based design cho testability
- Hỗ trợ multiple storage backends (S3, local file system)

#### **Service Layer Pattern**:
- Business logic được encapsulate trong service layer
- Services không phụ thuộc vào HTTP layer
- Transaction management ở service level

### 1.3. Design Principles Áp dụng

- **SOLID Principles**:
  - **S**ingle Responsibility: Mỗi package có một trách nhiệm rõ ràng
  - **O**pen/Closed: Dễ mở rộng qua interface, không cần sửa code cũ
  - **L**iskov Substitution: Repository interfaces có thể thay thế được
  - **I**nterface Segregation: Interfaces nhỏ, focused
  - **D**ependency Inversion: Depend on abstractions (interfaces), not concretions

- **DRY (Don't Repeat Yourself)**: BaseHandler, utils packages
- **KISS (Keep It Simple)**: Code straightforward, dễ hiểu
- **Separation of Concerns**: Handler → Service → Repository

### 1.4. Project Type

- **Backend API Service** (RESTful)
- **Microservice-ready**: Có thể deploy độc lập, có health check, metrics
- **Stateless**: Không lưu state trong memory (trừ metrics)
- **Cloud-native**: Tích hợp với AWS S3, có thể chạy trên container (Docker)

---

## 2. Cấu trúc Folder

### 2.1. Cấu trúc Thư mục Đầy đủ

```
Golang/
├── cmd/                          # Application entry points
│   ├── migrate/                  # Migration tool entry point
│   │   └── main.go               # CLI tool để chạy database migrations
│   └── server/                   # HTTP server entry point
│       └── main.go               # Main server application
│
├── internal/                     # Private application code (không export ra ngoài)
│   ├── config/                   # Configuration management
│   │   ├── config.go             # Config structs và Load()
│   │   ├── builder.go            # ConfigBuilder pattern
│   │   └── errors.go             # Config-specific errors
│   │
│   ├── container/                # Dependency Injection Container
│   │   └── container.go          # DI container, wire dependencies
│   │
│   ├── database/                 # Database layer
│   │   ├── database.go           # DB connection, transaction wrapper
│   │   └── migrations/           # SQL migration files
│   │       └── 001_create_uploads_table.sql
│   │
│   ├── handlers/                 # HTTP handlers (Presentation Layer)
│   │   ├── base_handler.go       # BaseHandler với common methods
│   │   ├── api_handler.go        # REST API handlers (upload, health)
│   │   ├── wp_handler.go         # WordPress-specific handlers
│   │   ├── metrics_handler.go    # Metrics endpoint handler
│   │   └── error_mapper.go       # Error mapping utilities
│   │
│   ├── middleware/               # HTTP middleware chain
│   │   ├── request_id.go         # Request ID middleware (tracing)
│   │   ├── logging.go            # Request/response logging
│   │   ├── cors.go               # CORS middleware
│   │   ├── csrf.go               # CSRF protection
│   │   ├── ratelimit.go          # Rate limiting (token bucket)
│   │   ├── concurrency.go        # Concurrency limiting (semaphore)
│   │   └── apikey.go             # API key authentication
│   │
│   ├── models/                   # Domain models / entities
│   │   ├── upload.go             # UploadResponse model
│   │   └── upload_record.go      # UploadRecord model (DB entity)
│   │
│   ├── repository/               # Data access layer (Repository Pattern)
│   │   ├── factory.go            # Repository factory
│   │   ├── errors.go             # Repository-specific errors
│   │   ├── s3_repository.go      # S3 storage repository
│   │   └── upload_repository.go  # Database repository (PostgreSQL)
│   │
│   ├── service/                  # Business logic layer
│   │   ├── factory.go            # Service factory
│   │   ├── errors.go             # Service-specific errors
│   │   ├── constants.go          # Service constants
│   │   ├── upload_service.go     # Upload business logic
│   │   ├── image_resize.go       # Image resize service (WordPress)
│   │   └── image_optimizer.go    # Image optimization service
│   │
│   ├── utils/                    # Utility functions
│   │   ├── file_utils.go         # File operations
│   │   ├── filename.go           # Filename sanitization
│   │   ├── file_size.go          # File size utilities
│   │   ├── content_validation.go # Content type validation
│   │   ├── multipart.go          # Multipart form handling
│   │   └── date_utils.go         # Date/time utilities
│   │
│   └── metrics/                  # Application metrics
│       └── metrics.go            # Metrics collection (in-memory)
│
├── docs/                         # Documentation
│   └── ARCHITECTURE.md           # This file
│
├── go.mod                        # Go module definition
├── go.sum                        # Go module checksums
│
└── [Binary files]                # Compiled binaries (if present)
    ├── server.exe                # Windows server binary
    └── migrate.exe               # Windows migrate binary
```

### 2.2. Chi tiết từng Thư mục

#### **`/cmd`** - Application Entry Points
- **Chức năng**: Chứa các main entry points cho application
- **Best Practice**: Mỗi binary có một thư mục riêng
- **Thành phần**:
  - `server/main.go`: Khởi tạo container, start HTTP server, graceful shutdown
  - `migrate/main.go`: CLI tool để chạy database migrations (up, status, version)

#### **`/internal`** - Private Application Code
- **Chức năng**: Code nội bộ, không export ra ngoài module (Go package visibility)
- **Best Practice**: Prevent external dependencies on internal implementation
- **Lưu ý**: Code trong `/internal` không thể import bởi projects khác

#### **`/internal/config`** - Configuration Management
- **Chức năng**: Quản lý configuration từ environment variables
- **Pattern**: Builder Pattern (`ConfigBuilder`)
- **Thành phần**:
  - `config.go`: Config structs
  - `builder.go`: ConfigBuilder với fluent API
  - `errors.go`: Config validation errors

#### **`/internal/container`** - Dependency Injection
- **Chức năng**: DI Container, wire tất cả dependencies
- **Pattern**: Container Pattern, Factory Pattern
- **Thành phần**:
  - Khởi tạo database, S3 client, repositories, services, handlers
  - Setup middleware chain
  - Graceful shutdown management

#### **`/internal/database`** - Database Layer
- **Chức năng**: Database connection, transaction management
- **Thành phần**:
  - `database.go`: DB wrapper, transaction support
  - `migrations/`: SQL migration files

#### **`/internal/handlers`** - HTTP Handlers (Presentation Layer)
- **Chức năng**: Xử lý HTTP requests/responses
- **Pattern**: Handler Pattern, BaseHandler với inheritance-like pattern
- **Thành phần**:
  - `base_handler.go`: Common response methods (SendSuccess, SendError)
  - `api_handler.go`: `/api/v1/upload`, `/api/v1/upload-transaction`, `/api/v1/health`
  - `wp_handler.go`: `/api/v1/wp-upload` (WordPress-specific)

#### **`/internal/middleware`** - HTTP Middleware
- **Chức năng**: Cross-cutting concerns (logging, auth, rate limiting)
- **Pattern**: Middleware Pattern, Chain of Responsibility
- **Thành phần**:
  - Request ID (tracing)
  - Logging
  - CORS
  - CSRF
  - Rate Limiting (in-memory token bucket)
  - Concurrency Limiting (semaphore)
  - API Key Authentication

#### **`/internal/models`** - Domain Models
- **Chức năng**: Domain entities và DTOs
- **Thành phần**:
  - `upload.go`: UploadResponse DTO
  - `upload_record.go`: UploadRecord entity (mapping với DB)

#### **`/internal/repository`** - Data Access Layer
- **Chức năng**: Tách biệt data access logic
- **Pattern**: Repository Pattern, Interface-based design
- **Thành phần**:
  - `s3_repository.go`: S3 storage operations
  - `upload_repository.go`: Database operations (prepared statements)
  - `factory.go`: Repository factory

#### **`/internal/service`** - Business Logic Layer
- **Chức năng**: Business logic, orchestration
- **Pattern**: Service Layer Pattern
- **Thành phần**:
  - `upload_service.go`: Upload business logic, transaction management
  - `image_resize.go`: Image resize logic (WordPress)
  - `image_optimizer.go`: Image optimization (JPEG/PNG/WebP)
  - `factory.go`: Service factory

#### **`/internal/utils`** - Utility Functions
- **Chức năng**: Reusable utility functions
- **Thành phần**: File operations, validation, formatting

#### **`/internal/metrics`** - Metrics Collection
- **Chức năng**: In-memory metrics collection
- **Thành phần**: Request metrics, upload metrics, image processing metrics

---

## 3. Công nghệ và Package

### 3.1. Ngôn ngữ & Version

- **Go (Golang)**: `1.24.0` (toolchain `go1.24.2`)
- **Module Name**: `s3-upload-tool`

### 3.2. Core Dependencies

#### **AWS SDK (v2)**
```
github.com/aws/aws-sdk-go-v2 v1.24.0
github.com/aws/aws-sdk-go-v2/config v1.26.1
github.com/aws/aws-sdk-go-v2/feature/s3/manager v1.15.7
github.com/aws/aws-sdk-go-v2/service/s3 v1.47.5
```
- **Vai trò**: Tương tác với Amazon S3
- **Lý do dùng**: Official AWS SDK, hỗ trợ credential chain (env vars, IAM roles, credentials file)
- **Tính năng sử dụng**:
  - S3 PutObject (upload)
  - Presigned URLs
  - Multipart upload (through manager.Uploader)

#### **Database Driver**
```
github.com/lib/pq v1.10.9
```
- **Vai trò**: PostgreSQL driver cho Go `database/sql`
- **Lý do dùng**: Standard PostgreSQL driver, hỗ trợ prepared statements, transactions

#### **Image Processing**
```
golang.org/x/image v0.34.0
```
- **Vai trò**: Image encoding/decoding, manipulation
- **Tính năng sử dụng**:
  - JPEG, PNG, GIF, WebP support
  - Image resize algorithms

#### **Utilities**
```
github.com/google/uuid v1.6.0
github.com/joho/godotenv v1.5.1
golang.org/x/sync v0.19.0
```
- **uuid**: Generate unique identifiers cho S3 keys
- **godotenv**: Load `.env` files
- **sync**: Synchronization primitives (mutex, sync.Once)

### 3.3. Standard Library Packages Sử dụng

- `net/http`: HTTP server, handlers, middleware
- `database/sql`: Database abstraction
- `context`: Context propagation, cancellation, timeouts
- `os`: File operations, environment variables
- `path/filepath`: Path manipulation
- `time`: Time utilities, timeouts
- `sync`: Synchronization (mutex, sync.Once)
- `encoding/json`: JSON encoding/decoding

---

## 4. Vòng đời Hoạt động

### 4.1. Bootstrapping (Application Startup)

#### **Step 1: Entry Point** (`cmd/server/main.go`)
```go
func main() {
    // 1. Initialize container (DI)
    ctn, err := container.NewContainer()
    
    // 2. Create HTTP server
    server := &http.Server{
        Addr: ":" + ctn.Config.Server.Port,
        Handler: ctn.GetServerHandler(),
    }
    
    // 3. Start server in goroutine
    go server.ListenAndServe()
    
    // 4. Wait for shutdown signal
    <-quit
}
```

#### **Step 2: Container Initialization** (`internal/container/container.go`)

```go
func NewContainer() (*Container, error) {
    // 1. Load configuration từ environment
    cfg, err := config.Load()
    
    // 2. Create directories (uploads, wp-uploads)
    os.MkdirAll(cfg.Directories.UploadDir, 0755)
    
    // 3. Initialize AWS S3 client (credential chain)
    awsCfg, err := awsconfig.LoadDefaultConfig(...)
    s3Client := s3.NewFromConfig(awsCfg)
    uploader := manager.NewUploader(s3Client)
    
    // 4. Initialize database (if enabled)
    db, err := database.NewDB(dbConfig)
    uploadRepo, err := repository.NewUploadRepository(db)
    
    // 5. Create repositories (Factory Pattern)
    repoFactory := repository.NewRepositoryFactory()
    s3Repo := repoFactory.CreateS3Repository(...)
    
    // 6. Create services (Factory Pattern)
    serviceFactory := service.NewServiceFactory(...)
    uploadService := serviceFactory.CreateUploadService(...)
    
    // 7. Create handlers
    apiHandler := handlers.NewAPIHandler(...)
    wpHandler := handlers.NewWPHandler(...) // if WordPress enabled
    
    return &Container{...}
}
```

#### **Step 3: Configuration Loading** (`internal/config/builder.go`)

```go
func (b *ConfigBuilder) BuildFromEnv() (*Config, error) {
    // 1. Load .env file (if exists)
    LoadEnvFile()
    
    // 2. Read environment variables với defaults
    port := getEnv("PORT", "8080")
    awsRegion := getEnv("AWS_REGION", "us-east-1")
    // ... more configs
    
    // 3. Build config với validation
    return NewConfigBuilder().
        WithServer(...).
        WithAWS(...).
        WithDatabase(...).
        Build()
}
```

#### **Step 4: Middleware Chain Setup** (`container.GetServerHandler()`)

Middleware được apply theo thứ tự (outermost → innermost):
1. **Concurrency Limiting** (nếu enabled)
2. **Rate Limiting** (nếu enabled)
3. **CSRF Protection** (nếu enabled)
4. **Metrics Middleware**
5. **Logging Middleware**
6. **Request ID Middleware** (innermost)

API-specific middleware:
- **CORS Middleware** (nếu CORS origins configured)
- **API Key Auth** (nếu require API key)

### 4.2. Request Processing Pipeline

1. **HTTP Request** → Server
2. **Middleware Chain** (concurrency → rate limit → CSRF → metrics → logging → request ID)
3. **Router** (mux.HandleFunc) → Handler
4. **Handler** → Service
5. **Service** → Repository
6. **Repository** → Database/S3
7. **Response** ← Handler (JSON)
8. **Middleware Chain** (reverse, logging/metrics)
9. **HTTP Response** ← Client

### 4.3. Graceful Shutdown

```go
// Signal handling
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

// Shutdown sequence:
// 1. Stop rate limiters (cleanup goroutines)
ctn.Shutdown()

// 2. Close database connections
db.Close()

// 3. Server shutdown với timeout
ctx, cancel := context.WithTimeout(..., ctn.Config.Server.ShutdownTimeout)
server.Shutdown(ctx)
```

### 4.4. Background Operations

- **Rate Limiter Cleanup**: Chạy định kỳ (default 10 phút) để cleanup expired tokens
- **Metrics Collection**: In-memory, không có background cleanup (có limit để tránh memory leak)

---

## 5. Luồng chảy Xử lý từ URL

### 5.1. Request Flow Diagram

```
HTTP Request
    ↓
[Concurrency Limiter Middleware]
    ↓
[Rate Limiter Middleware]
    ↓
[CSRF Middleware]
    ↓
[Metrics Middleware]
    ↓
[Logging Middleware]
    ↓
[Request ID Middleware]
    ↓
[Router] → Handler
    ↓
[Handler] → Service
    ↓
[Service] → Repository
    ↓
[Repository] → Database/S3
    ↓
Response ← Handler (JSON)
```

### 5.2. Chi tiết từng Bước

#### **Bước 1: HTTP Router** (`container.GetServerHandler()`)

**File**: `internal/container/container.go:205-299`

```go
func (c *Container) GetServerHandler() http.Handler {
    mux := http.NewServeMux()
    
    // API routes
    apiMux := http.NewServeMux()
    apiMux.HandleFunc("/api/v1/upload", c.APIHandler.HandleUpload)
    apiMux.HandleFunc("/api/v1/upload-transaction", c.APIHandler.HandleUploadWithTransaction)
    apiMux.HandleFunc("/api/v1/health", c.APIHandler.HandleHealth)
    apiMux.HandleFunc("/api/v1/metrics", metricsHandler.HandleMetrics)
    apiMux.HandleFunc("/api/v1/wp-upload", c.WPHandler.HandleWPUpload) // if WP enabled
    
    // Apply API middleware
    apiHandler := middleware.CORS(...)(apiMux)
    apiHandler = middleware.APIKeyAuth(...)(apiHandler)
    
    mux.Handle("/api/", apiHandler)
    
    // Root endpoint
    mux.HandleFunc("/", func(...) { ... })
    
    return handler // với global middleware
}
```

**Nhiệm vụ**:
- Route requests đến đúng handler
- Apply middleware cho từng route group

#### **Bước 2: Handler** (`internal/handlers/api_handler.go`)

**File**: `internal/handlers/api_handler.go:133-178`

**Ví dụ**: `POST /api/v1/upload`

```go
func (h *APIHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
    // 1. Create context với timeout
    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
    defer cancel()
    
    // 2. Parse multipart form
    reqData, err := h.processUploadRequest(r)
    // - ParseMultipartForm(maxSize)
    // - GetFileFromRequest()
    // - ValidateUploadFile()
    
    // 3. Call service
    result, err := h.uploadService.UploadImage(
        ctx,
        reqData.Header.Filename,
        reqData.File,
        reqData.Header.Size,
        h.maxUploadSize,
    )
    
    // 4. Record metrics
    metrics.GetMetrics().RecordUpload(err == nil, uploadDuration)
    
    // 5. Send response
    if err != nil {
        h.handleUploadError(w, err)
        return
    }
    
    h.SendSuccess(w, UploadResponseData{
        URL:  result.URL,
        Key:  result.Key,
        Size: reqData.Header.Size,
        Name: reqData.Header.Filename,
    })
}
```

**Nhiệm vụ**:
- Parse HTTP request (multipart form)
- Validate input
- Call service layer
- Map service errors → HTTP errors
- Format JSON response

#### **Bước 3: Service Layer** (`internal/service/upload_service.go`)

**File**: `internal/service/upload_service.go:53-202`

```go
func (s *uploadService) UploadImage(ctx context.Context, filename string, file io.Reader, fileSize int64, maxSize int64) (*models.UploadResponse, error) {
    // 1. Validate file type
    if !utils.IsAllowedFileType(filename) {
        return nil, ErrInvalidFileFormat
    }
    
    // 2. Validate file size
    if err := utils.ValidateFileSizeFromHeader(fileSize, maxSize); err != nil {
        return nil, NewFileSizeError(...)
    }
    
    // 3. Create temp file
    tempFile, err := os.CreateTemp(s.uploadDir, "upload-*"+filepath.Ext(filename))
    
    // 4. Copy file to temp (với timeout)
    bytesWritten, err := io.Copy(tempFile, io.LimitReader(file, maxSize+1))
    
    // 5. Validate actual size
    if bytesWritten > maxSize {
        return nil, NewFileSizeError(...)
    }
    
    // 6. Generate S3 key
    key := utils.GenerateS3Key(filename)
    contentType := utils.GetContentType(filename)
    
    // 7. Upload to S3 (Repository)
    _, err := s.s3Repo.Upload(ctx, s.bucketName, key, tempFile, contentType, s.useACL)
    
    // 8. Generate URL (presigned or public)
    var url string
    if s.usePresignedURL {
        url, err = s.s3Repo.GeneratePresignedURL(ctx, s.bucketName, key, expiry)
    } else {
        url = fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", ...)
    }
    
    return &models.UploadResponse{URL: url, Key: key}, nil
}
```

**Nhiệm vụ**:
- Business logic (validation, orchestration)
- Transaction management (cho upload với transaction)
- Error handling và mapping
- Orchestrate repository calls

#### **Bước 4: Repository Layer** (`internal/repository/s3_repository.go`)

**File**: `internal/repository/s3_repository.go:35-53`

```go
func (r *s3Repository) Upload(ctx context.Context, bucket, key string, body io.Reader, contentType string, useACL bool) (string, error) {
    input := &s3.PutObjectInput{
        Bucket:      aws.String(bucket),
        Key:         aws.String(key),
        Body:        body,
        ContentType: aws.String(contentType),
    }
    
    if useACL {
        input.ACL = types.ObjectCannedACLPublicRead
    }
    
    result, err := r.uploader.Upload(ctx, input)
    if err != nil {
        return "", fmt.Errorf("failed to upload to S3: %w", err)
    }
    
    return result.Location, nil
}
```

**Nhiệm vụ**:
- Data access logic (S3 operations)
- Map domain errors → repository errors
- Abstract storage implementation

#### **Bước 5: Response Flow**

1. **Repository** → returns (result, error)
2. **Service** → returns (UploadResponse, error)
3. **Handler** → maps error → HTTP status code
4. **Handler** → sends JSON response
5. **Middleware** → logs response, records metrics
6. **HTTP Response** → client

### 5.3. WordPress Upload Flow

**Endpoint**: `POST /api/v1/wp-upload`

**Flow**:
1. **Handler** (`wp_handler.go:55-166`):
   - Parse multipart form
   - Validate image file
   - Save to temp file
   
2. **ImageResizeService** (`image_resize.go`):
   - Save original image
   - Resize to multiple sizes (thumbnail, medium, large, ...)
   - Optimize images (JPEG/PNG quality, WebP conversion)
   - Save to WordPress-style directory structure (`/wp-content/uploads/YYYY/MM/`)
   
3. **Response**: Returns original + all resized sizes với URLs

---

## 6. Module / Feature Map

### 6.1. Core Modules

#### **1. Upload Module**
- **Chức năng**: Upload file lên S3
- **Files**:
  - `handlers/api_handler.go`: `HandleUpload()`, `HandleUploadWithTransaction()`
  - `service/upload_service.go`: `UploadImage()`, `UploadImageWithTransaction()`
  - `repository/s3_repository.go`: `Upload()`
  - `repository/upload_repository.go`: Database operations
- **Endpoints**:
  - `POST /api/v1/upload`: Upload file (không transaction)
  - `POST /api/v1/upload-transaction`: Upload file với database transaction
- **Dependencies**: Config, AWS S3, Database (optional)

#### **2. WordPress Integration Module**
- **Chức năng**: Upload và resize images cho WordPress
- **Files**:
  - `handlers/wp_handler.go`: `HandleWPUpload()`
  - `service/image_resize.go`: `ResizeImage()`, `SaveOriginal()`
  - `service/image_optimizer.go`: `OptimizeImage()`
- **Endpoints**:
  - `POST /api/v1/wp-upload`: Upload image với resize
  - `GET /wp-content/uploads/**`: Serve uploaded images
- **Dependencies**: Upload Module, Image processing

#### **3. Health Check Module**
- **Chức năng**: Health check endpoint
- **Files**:
  - `handlers/api_handler.go`: `HandleHealth()`
- **Endpoints**:
  - `GET /api/v1/health`: Health check (database, S3 connectivity, disk space)
- **Dependencies**: Database, S3 Repository

#### **4. Metrics Module**
- **Chức năng**: Application metrics collection
- **Files**:
  - `metrics/metrics.go`: Metrics collection
  - `handlers/metrics_handler.go`: Metrics endpoint
  - `middleware/metrics.go`: Metrics middleware
- **Endpoints**:
  - `GET /api/v1/metrics`: Metrics endpoint
- **Dependencies**: None (in-memory)

#### **5. Security Module**
- **Chức năng**: Security middleware
- **Files**:
  - `middleware/csrf.go`: CSRF protection
  - `middleware/apikey.go`: API key authentication
  - `middleware/cors.go`: CORS handling
  - `middleware/ratelimit.go`: Rate limiting
  - `middleware/concurrency.go`: Concurrency limiting
- **Dependencies**: Config

#### **6. Database Module**
- **Chức năng**: Database operations
- **Files**:
  - `database/database.go`: Connection, transactions
  - `repository/upload_repository.go`: Upload records CRUD
  - `models/upload_record.go`: UploadRecord entity
  - `database/migrations/*.sql`: Migration files
- **Dependencies**: PostgreSQL driver

### 6.2. Module Dependencies

```
┌─────────────────┐
│   Upload Module │
└────────┬────────┘
         │
         ├─→ S3 Repository
         ├─→ Upload Repository (optional)
         └─→ Config
         
┌─────────────────┐
│ WordPress Module│
└────────┬────────┘
         │
         ├─→ Image Resize Service
         ├─→ Image Optimizer
         └─→ Upload Module
         
┌─────────────────┐
│  Health Module  │
└────────┬────────┘
         │
         ├─→ Database
         └─→ S3 Repository
         
┌─────────────────┐
│ Security Module │
└────────┬────────┘
         │
         └─→ Config
```

---

## 7. Data Flow (Backend)

### 7.1. Upload Request Flow

#### **Request**:
```http
POST /api/v1/upload
Content-Type: multipart/form-data

file: [binary data]
```

#### **Response**:
```json
{
  "success": true,
  "data": {
    "url": "https://bucket.s3.region.amazonaws.com/key",
    "key": "uploads/2024/01/uuid-filename.jpg",
    "size": 1024000,
    "name": "image.jpg"
  }
}
```

#### **Error Response**:
```json
{
  "success": false,
  "error": {
    "code": "FILE_TOO_LARGE",
    "message": "File exceeds maximum size: 2MB"
  }
}
```

### 7.2. Upload with Transaction Flow

#### **Request**: Tương tự upload request

#### **Response**:
```json
{
  "success": true,
  "data": {
    "url": "https://bucket.s3.region.amazonaws.com/key",
    "key": "uploads/2024/01/uuid-filename.jpg",
    "size": 1024000,
    "name": "image.jpg",
    "record": {
      "id": 123,
      "filename": "uuid-filename.jpg",
      "original_name": "image.jpg",
      "file_size": 1024000,
      "content_type": "image/jpeg",
      "s3_key": "uploads/2024/01/uuid-filename.jpg",
      "s3_url": "https://bucket.s3.region.amazonaws.com/key",
      "status": "completed",
      "created_at": "2024-01-15T10:30:00Z",
      "updated_at": "2024-01-15T10:30:05Z"
    }
  }
}
```

### 7.3. WordPress Upload Flow

#### **Request**:
```http
POST /api/v1/wp-upload
Content-Type: multipart/form-data

file: [image binary]
```

#### **Response**:
```json
{
  "success": true,
  "data": {
    "file": {
      "name": "image.jpg",
      "type": "image/jpeg",
      "url": "https://example.com/wp-content/uploads/2024/01/image.jpg",
      "size": 1024000
    },
    "sizes": [
      {
        "name": "thumbnail",
        "file": "image-150x150.jpg",
        "width": 150,
        "height": 150,
        "url": "https://example.com/wp-content/uploads/2024/01/image-150x150.jpg"
      },
      {
        "name": "medium",
        "file": "image-300x300.jpg",
        "width": 300,
        "height": 300,
        "url": "https://example.com/wp-content/uploads/2024/01/image-300x300.jpg"
      }
    ]
  }
}
```

### 7.4. Health Check Response

```json
{
  "success": true,
  "data": {
    "status": "ok",
    "service": "s3-upload-api",
    "max_size": 2097152,
    "max_size_formatted": "2 MB",
    "request_id": "abc123",
    "checks": {
      "disk": {
        "status": "ok"
      },
      "database": {
        "status": "ok",
        "open_connections": 5,
        "in_use": 2,
        "idle": 3,
        "wait_count": 0
      },
      "s3_service": {
        "status": "ok"
      }
    }
  }
}
```

### 7.5. Metrics Response

```json
{
  "success": true,
  "data": {
    "requests": {
      "/api/v1/upload": {
        "count": 150,
        "error_count": 5,
        "avg_duration_ms": 1250
      },
      "/api/v1/health": {
        "count": 500,
        "error_count": 0,
        "avg_duration_ms": 50
      }
    },
    "uploads": {
      "total": 150,
      "success": 145,
      "failed": 5,
      "avg_duration_ms": 1200
    },
    "image_processing": {
      "resize_count": 450,
      "optimize_count": 450,
      "optimize_saved_bytes": 52428800
    }
  }
}
```

---

## 8. Entity & Database Overview

### 8.1. Database Schema

#### **Table: `uploads`**

```sql
CREATE TABLE uploads (
    id BIGSERIAL PRIMARY KEY,
    filename VARCHAR(255) NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    s3_key VARCHAR(500),
    s3_url VARCHAR(1000),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_uploads_status ON uploads (status);
CREATE INDEX idx_uploads_created_at ON uploads (created_at DESC);
CREATE INDEX idx_uploads_s3_key ON uploads (s3_key);
```

#### **Table: `schema_migrations`** (tự động tạo bởi migrate tool)

```sql
CREATE TABLE schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### 8.2. Entity Models

#### **UploadRecord** (`internal/models/upload_record.go`)

```go
type UploadRecord struct {
    ID           int64     `db:"id"`
    Filename     string    `db:"filename"`
    OriginalName string    `db:"original_name"`
    FileSize     int64     `db:"file_size"`
    ContentType  string    `db:"content_type"`
    S3Key        string    `db:"s3_key"`
    S3URL        string    `db:"s3_url"`
    Status       string    `db:"status"` // "pending", "completed", "failed"
    Error        *string   `db:"error"`
    CreatedAt    time.Time `db:"created_at"`
    UpdatedAt    time.Time `db:"updated_at"`
}
```

**Upload Status Enum**:
- `pending`: Đang xử lý
- `completed`: Hoàn thành
- `failed`: Thất bại

#### **UploadResponse** (`internal/models/upload.go`)

```go
type UploadResponse struct {
    URL string // S3 URL hoặc presigned URL
    Key string // S3 key
}
```

### 8.3. Database Operations

#### **CreateUpload** (`repository/upload_repository.go:79-113`)
- Insert upload record vào database
- Chạy trong transaction
- Return created record với ID

#### **UpdateUploadStatus** (`repository/upload_repository.go:115-145`)
- Update status, S3 key, S3 URL, error message
- Chạy trong transaction
- Validate rows affected

#### **GetUploadByID** (`repository/upload_repository.go:147-177`)
- Query upload record theo ID
- Dùng prepared statement
- Return error nếu không tìm thấy

#### **GetUploadsByStatus** (`repository/upload_repository.go:179-220`)
- Query uploads theo status
- Support pagination (limit)
- Dùng prepared statement

### 8.4. Transaction Management

Transaction được quản lý ở **Service Layer**:

```go
func (s *uploadService) UploadImageWithTransaction(...) {
    // 1. Begin transaction
    tx, err := s.db.BeginTx(ctx, nil)
    defer func() {
        if err != nil {
            tx.Rollback()
        }
    }()
    
    // 2. Create upload record
    record, err := s.uploadRepo.CreateUpload(ctx, tx, uploadRecord)
    
    // 3. Upload to S3
    _, err = s.s3Repo.Upload(...)
    
    // 4. Update upload status
    err = s.uploadRepo.UpdateUploadStatus(ctx, tx, record.ID, ...)
    
    // 5. Commit transaction
    err = tx.Commit()
}
```

**Lưu ý**:
- Transaction chỉ dùng cho database operations
- S3 upload không thể rollback (cần manual cleanup nếu fail)
- Transaction timeout được handle bởi context

---

## 9. Summary

### 9.1. Kiến trúc Tổng quan

**S3 Upload Tool** là một **RESTful API service** được xây dựng với **Clean Architecture**, áp dụng các design patterns phổ biến:

- **Layered Architecture**: Handler → Service → Repository
- **Dependency Injection**: Container pattern
- **Repository Pattern**: Tách biệt data access
- **Service Layer Pattern**: Business logic encapsulation
- **Factory Pattern**: Repository và Service factories
- **Middleware Pattern**: Cross-cutting concerns

### 9.2. Điểm Mạnh

✅ **Clean Code**: Code rõ ràng, dễ đọc, dễ maintain  
✅ **Separation of Concerns**: Mỗi layer có trách nhiệm rõ ràng  
✅ **Testability**: Interface-based design, dễ mock/test  
✅ **Security**: CSRF, CORS, API key auth, rate limiting, concurrency limiting  
✅ **Scalability**: Stateless design, có thể scale horizontal  
✅ **Observability**: Metrics, logging, request ID tracing  
✅ **Error Handling**: Structured errors, proper HTTP status codes  
✅ **Transaction Support**: Database transaction cho upload tracking  

### 9.3. Công nghệ Stack

- **Language**: Go 1.24.0
- **Database**: PostgreSQL (optional)
- **Storage**: Amazon S3
- **Image Processing**: golang.org/x/image
- **HTTP Server**: net/http (standard library)

### 9.4. Tính năng Chính

1. **File Upload**: Upload file lên S3 với validation, timeout handling
2. **Upload Tracking**: Database tracking với transaction support
3. **WordPress Integration**: Image resize, optimization
4. **Security**: CSRF, CORS, API key auth, rate/concurrency limiting
5. **Monitoring**: Health check, metrics endpoint
6. **Migration Tool**: Database migration CLI tool

### 9.5. Best Practices Áp dụng

✅ **SOLID Principles**  
✅ **Error Wrapping**: `fmt.Errorf("...: %w", err)`  
✅ **Context Propagation**: Tất cả I/O operations dùng context  
✅ **Prepared Statements**: SQL injection prevention  
✅ **Graceful Shutdown**: Cleanup resources  
✅ **Timeout Management**: Context với timeout cho tất cả operations  
✅ **Structured Logging**: Log với context (request ID)  
✅ **Security**: Input validation, output sanitization  

### 9.6. Hướng Phát triển

🚀 **Potential Improvements**:
- Add unit tests và integration tests
- Add OpenAPI/Swagger documentation
- Add distributed tracing (Jaeger, Zipkin)
- Add caching layer (Redis) cho metrics
- Add queue system cho async upload processing
- Add WebSocket support cho real-time upload progress
- Add multi-tenant support
- Add file versioning
- Add CDN integration (CloudFront)

---

**Tài liệu được tạo**: 2024  
**Version**: 1.0  
**Maintainer**: Development Team

