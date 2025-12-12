# Phân Tích Chi Tiết Toàn Bộ Sourcecode

## 📋 Tổng Quan

Tài liệu này phân tích **toàn bộ sourcecode** của project S3 Upload Tool từ mọi góc độ: kiến trúc, patterns, security, performance, error handling, và các chi tiết kỹ thuật.

---

## 1. KIẾN TRÚC & DESIGN PATTERNS

### 1.1. Layered Architecture

Project áp dụng **Clean Architecture** với 3 lớp rõ ràng:

```
┌─────────────────────────────────────────┐
│   Presentation Layer (Handlers)         │
│   - api_handler.go                      │
│   - wp_handler.go                       │
│   - base_handler.go                     │
│   - error_mapper.go                     │
└─────────────────────────────────────────┘
                  ↓
┌─────────────────────────────────────────┐
│   Business Logic Layer (Services)       │
│   - upload_service.go                   │
│   - image_resize.go                     │
│   - image_optimizer.go                  │
└─────────────────────────────────────────┘
                  ↓
┌─────────────────────────────────────────┐
│   Data Access Layer (Repositories)      │
│   - s3_repository.go                    │
│   - upload_repository.go                │
│   - database.go                         │
└─────────────────────────────────────────┘
```

### 1.2. Design Patterns Được Áp Dụng

#### **1. Dependency Injection (Container Pattern)**
**File**: `internal/container/container.go`

```go
type Container struct {
    Config           *config.Config
    DB               *database.DB
    S3Client         *s3.Client
    Uploader         *manager.Uploader
    S3Repository     repository.S3Repository
    UploadRepository repository.UploadRepository
    UploadService    service.UploadService
    APIHandler       *handlers.APIHandler
    WPHandler        *handlers.WPHandler
    RepoFactory      *repository.RepositoryFactory
    ServiceFactory   *service.ServiceFactory
    RateLimiters     []*middleware.RateLimiter
}
```

**Đặc điểm**:
- Centralized dependency management
- Constructor injection
- Lifecycle management (Shutdown method)
- Factory pattern integration

#### **2. Repository Pattern**
**Files**: `internal/repository/s3_repository.go`, `internal/repository/upload_repository.go`

**Interface-based design**:
```go
type S3Repository interface {
    Upload(ctx, bucket, key, body, contentType, useACL) (string, error)
    GeneratePresignedURL(ctx, bucket, key, expiry) (string, error)
    CheckConnectivity(ctx) error
}

type UploadRepository interface {
    CreateUpload(ctx, tx, record) (*UploadRecord, error)
    UpdateUploadStatus(ctx, tx, id, status, s3Key, s3URL, err) error
    GetUploadByID(ctx, db, id) (*UploadRecord, error)
    GetUploadsByStatus(ctx, db, status, limit) ([]*UploadRecord, error)
}
```

**Lợi ích**:
- Tách biệt data access logic
- Dễ test (mock interfaces)
- Dễ thay đổi implementation (S3 → Local, PostgreSQL → MySQL)
- Transaction support qua `*database.Tx`

#### **3. Factory Pattern**
**Files**: `internal/repository/factory.go`, `internal/service/factory.go`

```go
type RepositoryFactory struct{}

func (f *RepositoryFactory) CreateRepository(repoType RepositoryType, ...) (S3Repository, error) {
    switch repoType {
    case RepositoryTypeS3:
        return f.CreateS3Repository(...), nil
    default:
        return nil, ErrUnsupportedRepositoryType
    }
}
```

**Lợi ích**:
- Encapsulate object creation
- Dễ mở rộng (thêm repository types mới)
- Consistent creation logic

#### **4. Service Layer Pattern**
**File**: `internal/service/upload_service.go`

**Business logic encapsulation**:
- Validation logic (file type, size)
- Orchestration (temp file → S3 → DB)
- Transaction management
- Error handling và mapping

#### **5. Middleware Pattern (Chain of Responsibility)**
**Files**: `internal/middleware/*.go`

**Middleware chain** (outermost → innermost):
1. Concurrency Limiter
2. Rate Limiter
3. CSRF Protection
4. Metrics Collection
5. Logging
6. Request ID

**Implementation**:
```go
handler = middleware.RequestIDMiddleware(handler)
handler = middleware.LoggingMiddleware(handler)
handler = metrics.MetricsMiddleware(handler)
handler = csrfProtection.CSRFMiddleware(handler)
handler = middleware.RateLimitMiddleware(rateLimiter)(handler)
handler = middleware.ConcurrencyLimitMiddleware(concurrencyLimiter)(handler)
```

#### **6. Builder Pattern**
**File**: `internal/config/builder.go`

```go
func NewConfigBuilder() *ConfigBuilder

func (b *ConfigBuilder) WithServer(...) *ConfigBuilder
func (b *ConfigBuilder) WithAWS(...) *ConfigBuilder
func (b *ConfigBuilder) WithDatabase(...) *ConfigBuilder
// ... fluent API

func (b *ConfigBuilder) Build() (*Config, error)
```

**Lợi ích**:
- Fluent API dễ đọc
- Validation tại Build()
- Default values tự động

---

## 2. ERROR HANDLING STRATEGY

### 2.1. Error Wrapping (Go 1.13+)

**Pattern**: Dùng `fmt.Errorf("...: %w", err)` để wrap errors

**Ví dụ**:
```go
// internal/service/upload_service.go
return nil, fmt.Errorf("failed to upload to S3 bucket %s, key %s: %w", bucket, key, ErrUploadToS3)
```

**Lợi ích**:
- Preserve error chain
- Dùng `errors.Is()` và `errors.As()` để check
- Debug dễ dàng hơn

### 2.2. Custom Error Types

**File**: `internal/service/errors.go`

```go
type FileSizeError struct {
    ActualSize int64
    MaxSize    int64
    Message    string
}

func (e *FileSizeError) Error() string {
    if e.Message != "" {
        return e.Message
    }
    return fmt.Sprintf("file quá lớn: %d bytes (giới hạn: %d bytes)", e.ActualSize, e.MaxSize)
}
```

**Lợi ích**:
- Type-safe error checking
- Rich error context
- User-friendly messages

### 2.3. Error Mapping (Handler Layer)

**File**: `internal/handlers/error_mapper.go`

**Strategy**: Map internal errors → user-friendly HTTP errors

```go
type errorMapper struct {
    mappings map[string]string
}

func sanitizeError(err error) string {
    // ! Trả về message generic để tránh leak thông tin nội bộ
    return defaultErrorMapper.mapError(err)
}
```

**Đặc điểm**:
- Sanitize errors để không leak internal info
- Type checking cho custom errors (FileSizeError)
- Pattern matching cho generic errors

### 2.4. Error Propagation

**Flow**:
1. **Repository** → returns wrapped error với context
2. **Service** → wraps thêm business context
3. **Handler** → maps to HTTP status codes, sanitizes message
4. **Client** → nhận user-friendly error

**Ví dụ**:
```go
// Repository
return nil, fmt.Errorf("failed to create upload record: %w", err)

// Service
return nil, nil, fmt.Errorf("failed to create upload record: %w", err)

// Handler
if err != nil {
    h.handleUploadError(w, err) // → sanitize và map
}
```

---

## 3. SECURITY MEASURES

### 3.1. Input Validation & Sanitization

#### **Filename Sanitization**
**File**: `internal/utils/filename.go`

```go
func SanitizeFilename(filename string) (string, error) {
    // ! Xóa path traversal attempts
    filename = strings.ReplaceAll(filename, "..", "")
    filename = strings.ReplaceAll(filename, "/", "-")
    filename = strings.ReplaceAll(filename, "\\", "-")
    
    // ! Xóa control characters và các ký tự nguy hiểm
    // Remove: < > : " | ? *
    // ...
}
```

**Bảo vệ chống**:
- Path traversal (`../`)
- Directory traversal
- Control characters
- Dangerous characters

#### **Content Validation**
**File**: `internal/utils/content_validation.go`

```go
func ValidateFileContent(file multipart.File, filename string) error {
    // * Đọc 512 bytes đầu để detect MIME type
    buffer := make([]byte, 512)
    detectedType := http.DetectContentType(buffer[:n])
    expectedType := GetContentType(filename)
    
    // ! Validate image MIME types chặt chẽ hơn
    if IsImage(filename) {
        // Strict validation cho images
    }
}
```

**Bảo vệ chống**:
- MIME type spoofing
- File extension mismatch
- Malicious file uploads

#### **File Size Validation**
**File**: `internal/utils/file_size.go`, `internal/service/upload_service.go`

```go
// Double validation:
// 1. Header size check
if err := utils.ValidateFileSizeFromHeader(fileSize, maxSize); err != nil {
    return nil, NewFileSizeError(...)
}

// 2. Actual size check sau khi copy
if bytesWritten > maxSize {
    return nil, NewFileSizeError(...)
}

// 3. Detect truncation
if bytesWritten == maxSize+1 {
    return nil, NewFileSizeError(...)
}
```

### 3.2. SQL Injection Prevention

**File**: `internal/repository/upload_repository.go`

**Strategy**:
1. **Prepared Statements** cho non-transaction queries:
```go
getByIDStmt, err := db.Prepare(`
    SELECT ... FROM uploads WHERE id = $1
`)
```

2. **Parameterized Queries** cho transaction queries:
```go
err := tx.QueryRowContext(ctx, query,
    record.Filename,  // $1
    record.OriginalName, // $2
    // ... positional parameters
)
```

3. **SQL Identifier Quoting** (defense-in-depth):
```go
// cmd/migrate/main.go
tableName := pq.QuoteIdentifier(migrationsTableName)
```

**Kết quả**: Không có SQL injection vulnerabilities

### 3.3. CSRF Protection

**File**: `internal/middleware/csrf.go`

**Algorithm**: Double-Submit Cookie Pattern

**Flow**:
1. GET request → Generate CSRF token → Set cookie
2. POST/PUT/DELETE → Validate token từ cookie == token từ header/form
3. Tokens phải match chính xác

**Implementation**:
```go
func (c *CSRFProtection) validateCSRF(r *http.Request) bool {
    cookieToken := c.getCSRFCookie(r)
    requestToken := c.getCSRFToken(r) // từ X-CSRF-Token header hoặc form
    
    // * Tokens phải match chính xác (double-submit cookie pattern)
    return cookieToken == requestToken
}
```

**Bảo vệ chống**:
- Cross-Site Request Forgery
- Session fixation (cùng với SameSite cookie)

### 3.4. API Key Authentication

**File**: `internal/middleware/apikey.go`

**Security Measures**:
1. **Constant-time comparison** để tránh timing attacks:
```go
// ! Dùng constant-time comparison để tránh timing attacks
if subtle.ConstantTimeCompare([]byte(providedKey), []byte(apiKey)) != 1 {
    // reject
}
```

2. **Chỉ accept từ header**, không từ query params:
```go
// ! Chỉ check X-API-Key header (không bao giờ accept từ query parameters)
providedKey := r.Header.Get("X-API-Key")
```

**Bảo vệ chống**:
- Timing attacks
- API key leakage qua query params (logs, referrer)

### 3.5. Rate Limiting

**File**: `internal/middleware/ratelimit.go`

**Algorithm**: Token Bucket

**Implementation**:
- In-memory token bucket per IP
- Configurable rate và window
- Automatic token refill
- Cleanup goroutine để prevent memory leaks

**Features**:
- High concurrency optimized (batch deletion)
- Double-check pattern để tránh race conditions
- Context cancellation support

### 3.6. Concurrency Limiting

**File**: `internal/middleware/concurrency.go`

**Implementation**: Semaphore (weighted)

```go
sem := semaphore.NewWeighted(maxConcurrent)
err := sem.Acquire(ctx, 1)
defer sem.Release(1)
```

**Bảo vệ chống**:
- Resource exhaustion
- DoS attacks
- Server overload

### 3.7. CORS Protection

**File**: `internal/middleware/cors.go`

**Features**:
- Configurable allowed origins
- Preflight OPTIONS request handling
- Security warning cho wildcard (`*`) trong production

### 3.8. Request ID Tracing

**File**: `internal/middleware/request_id.go`

**Purpose**: Distributed tracing

**Flow**:
1. Check `X-Request-ID` header từ upstream proxy
2. Generate UUID nếu không có
3. Set vào response header và context
4. Logging middleware sử dụng để trace requests

### 3.9. AWS Credentials Security

**File**: `internal/config/config.go`, `internal/container/container.go`

**Strategy**: AWS SDK Credential Chain (không hardcode)

**Order**:
1. Environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`)
2. IAM role (EC2/ECS/Lambda)
3. AWS credentials file (`~/.aws/credentials`)
4. EC2 Instance Metadata Service

**Lợi ích**:
- Không hardcode credentials trong code
- Support multiple deployment scenarios
- Best practice AWS security

### 3.10. File Server Security

**File**: `internal/container/container.go:320-331`

```go
func secureFileServer(dir http.Dir) http.Handler {
    fs := http.FileServer(dir)
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Disable directory listing - return 404 for directory requests
        if strings.HasSuffix(r.URL.Path, "/") {
            http.NotFound(w, r)
            return
        }
        fs.ServeHTTP(w, r)
    })
}
```

**Bảo vệ chống**:
- Directory listing exposure
- Information disclosure

---

## 4. CONTEXT & TIMEOUT MANAGEMENT

### 4.1. Context Propagation

**Pattern**: Tất cả I/O operations đều nhận `context.Context`

**Ví dụ**:
```go
// Handler
ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
defer cancel()

// Service
func (s *uploadService) UploadImage(ctx context.Context, ...) {
    uploadCtx := ctx
    if s.uploadTimeout > 0 {
        uploadCtx, cancel = context.WithTimeout(ctx, s.uploadTimeout)
        defer cancel()
    }
}

// Repository
func (r *s3Repository) Upload(ctx context.Context, ...) {
    result, err := r.uploader.Upload(ctx, input)
}
```

**Lợi ích**:
- Cancellation propagation
- Timeout control
- Graceful shutdown

### 4.2. Timeout Strategy

**Multiple timeout layers**:

1. **Request Timeout** (Handler): 5 phút
2. **Upload Timeout** (Service): Configurable (default 60s)
3. **File Copy Timeout** (Service): 30s
4. **Context Check** (Service): Periodic cancellation checks

**Ví dụ**:
```go
// internal/service/upload_service.go
copyCtx, copyCancel := context.WithTimeout(uploadCtx, 30*time.Second)
defer copyCancel()

// io.Copy không hỗ trợ context cancellation
// ? Dùng goroutine + channel để implement timeout
done := make(chan error, 1)
go func() {
    bytesWritten, err = io.Copy(tempFile, limitedReader)
    done <- err
}()

select {
case err := <-done:
    // handle
case <-copyCtx.Done():
    // ! Đóng file để interrupt I/O
    tempFile.Close()
    return nil, fmt.Errorf("file copy timeout: %w", copyCtx.Err())
}
```

### 4.3. Cancellation Handling

**Pattern**: Periodic cancellation checks trong long-running operations

**Ví dụ**:
```go
// internal/service/image_resize.go
for i, sizeConfig := range s.sizes {
    // * Check context cancellation periodically (mỗi 5 sizes)
    if i%5 == 0 {
        select {
        case <-ctx.Done():
            return nil, fmt.Errorf("resize cancelled: %w", ctx.Err())
        default:
        }
    }
}
```

---

## 5. PERFORMANCE OPTIMIZATIONS

### 5.1. Database Connection Pooling

**File**: `internal/database/database.go`

```go
db.SetMaxOpenConns(cfg.MaxOpen)      // Default: 25
db.SetMaxIdleConns(cfg.MaxIdle)      // Default: 5
db.SetConnMaxLifetime(cfg.MaxLifetime) // Default: 5 minutes
```

**Tối ưu**:
- Connection reuse
- Prevent connection exhaustion
- Automatic stale connection cleanup

### 5.2. Prepared Statements

**File**: `internal/repository/upload_repository.go`

**Lợi ích**:
- Query parsing chỉ 1 lần
- Faster execution
- SQL injection prevention

**Lưu ý**: Chỉ dùng cho non-transaction queries (transaction queries chạy trong tx context)

### 5.3. Worker Pools cho Image Processing

**Files**: `internal/service/image_resize.go`, `internal/service/image_optimizer.go`

**Pattern**: Goroutine pool cho parallel processing

```go
// * Dùng worker pool cho parallel processing để improve performance
numWorkers := runtime.NumCPU()
height := bounds.Max.Y - bounds.Min.Y
if numWorkers > height {
    numWorkers = height
}

var wg sync.WaitGroup
rowChan := make(chan int, height)

// Start workers
for w := 0; w < numWorkers; w++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        for y := range rowChan {
            // Process row
        }
    }()
}

// Send work
go func() {
    defer close(rowChan)
    for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
        rowChan <- y
    }
}()

wg.Wait()
```

**Áp dụng cho**:
- Bilinear interpolation resize (image_resize.go)
- Color depth reduction (image_optimizer.go)

**Lợi ích**:
- CPU-bound tasks parallelized
- Optimal worker count (runtime.NumCPU())
- Context cancellation support

### 5.4. HTTP Client Connection Pooling

**File**: `internal/container/container.go:65-75`

```go
httpClient := &http.Client{
    Timeout: 60 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        500,
        MaxIdleConnsPerHost: 100,
        IdleConnTimeout:     90 * time.Second,
        DisableKeepAlives:   false,
        DisableCompression:  false,
    },
}
```

**Tối ưu cho**:
- High concurrency scenarios
- AWS SDK calls
- Connection reuse

### 5.5. S3 Multipart Upload Configuration

**File**: `internal/container/container.go:98-103`

```go
uploader := manager.NewUploader(s3Client, func(u *manager.Uploader) {
    u.PartSize = 10 * 1024 * 1024  // 10MB per part
    u.Concurrency = 5               // 5 concurrent parts
    u.LeavePartsOnError = false     // Cleanup on error
})
```

**Lợi ích**:
- Parallel upload parts
- Resumable uploads
- Better throughput cho large files

### 5.6. Image Optimization Strategy

**File**: `internal/service/image_optimizer.go`

**Multi-quality Testing**:
```go
// * Thử nhiều quality levels để tìm compression tốt nhất
baseQuality := o.calculateOptimalJPEGQuality(originalSize)
qualities := []int{baseQuality - 5, baseQuality, baseQuality + 5}

for i, quality := range qualities {
    // Encode với mỗi quality
    // So sánh file sizes
    // Chọn best result
}
```

**Adaptive Quality**:
```go
// * File lớn hơn → compression aggressive hơn
if fileSize > LargeFileThreshold {
    return max(MinJPEGQuality, baseQuality-LargeFileQualityReduction)
} else if fileSize > MediumFileThreshold {
    return max(MinJPEGQuality+5, baseQuality-MediumFileQualityReduction)
} else if fileSize < SmallFileThreshold {
    return min(MaxJPEGQuality, baseQuality+SmallFileQualityIncrease)
}
```

**Lợi ích**:
- Optimal file size vs quality balance
- Adaptive compression based on file size

### 5.7. Metrics Collection Optimization

**File**: `internal/metrics/metrics.go`

**Memory Management**:
- Limit tracked endpoints (maxEndpoints = 1000)
- Keep only last 100 durations per endpoint
- Slice copy để tránh race conditions

```go
// Create a copy of the slice to avoid race conditions
durations := make([]time.Duration, len(m.requestDuration[endpoint]))
copy(durations, m.requestDuration[endpoint])
```

---

## 6. CONCURRENCY & GOROUTINES

### 6.1. Concurrency Patterns

#### **Worker Pool Pattern**
- Image processing (resize, optimize)
- Parallel row processing

#### **Channel-based Communication**
- Work distribution (rowChan)
- Error reporting (done channel)
- Cancellation (context.Done())

#### **WaitGroup Synchronization**
- Wait for all workers to complete
- Proper cleanup

### 6.2. Goroutine Safety

**Mutex Protection**:
- Metrics collection (`sync.RWMutex`)
- Rate limiter (`sync.RWMutex` + visitor-level mutex)
- Concurrency limiter (`atomic` operations)

**Pattern**:
```go
// Metrics
m.mu.RLock()
defer m.mu.RUnlock()

// Rate Limiter (nested locks)
rl.mu.RLock()
v.mu.Lock()
// ... operations
v.mu.Unlock()
rl.mu.RUnlock()
```

### 6.3. Goroutine Cleanup

**Rate Limiter Cleanup**:
```go
// Background goroutine với context cancellation
go rl.cleanup()

func (rl *RateLimiter) cleanup() {
    ticker := time.NewTicker(rl.cleanupTick)
    defer ticker.Stop()
    
    for {
        select {
        case <-rl.ctx.Done():
            return // Stop cleanup goroutine
        case <-ticker.C:
            // Cleanup old visitors
        }
    }
}

// Shutdown
func (rl *RateLimiter) Stop() {
    rl.cancel()
}
```

**Container Shutdown**:
```go
func (c *Container) Shutdown() {
    // Stop all rate limiters to prevent goroutine leaks
    for _, rl := range c.RateLimiters {
        rl.Stop()
    }
}
```

---

## 7. TRANSACTION MANAGEMENT

### 7.1. Database Transactions

**File**: `internal/service/upload_service.go:204-333`

**Pattern**: Named return variables + defer rollback

```go
func (s *uploadService) UploadImageWithTransaction(...) 
    (resp *models.UploadResponse, record *models.UploadRecord, err error) {
    
    tx, err := s.db.BeginTx(ctx, nil)
    
    // ! Dùng named return variables để đảm bảo rollback đúng
    var txErr error
    defer func() {
        // ! Xử lý panic: rollback trước khi re-panic
        if p := recover(); p != nil {
            tx.Rollback()
            panic(p)
        }
        
        // ! Rollback nếu có bất kỳ lỗi nào
        if err != nil || txErr != nil {
            tx.Rollback()
        }
    }()
    
    // ... operations ...
    
    if err = tx.Commit(); err != nil {
        txErr = err
        return
    }
    
    return resp, record, nil
}
```

**Đặc điểm**:
- Panic-safe (rollback on panic)
- Error-safe (rollback on any error)
- Named returns để ensure rollback logic

### 7.2. Transaction Isolation

**Note**: Không explicit set isolation level (dùng database default)

**Transaction Flow**:
1. Begin transaction
2. Create upload record (status: pending)
3. Upload to S3
4. Update upload record (status: completed, s3_key, s3_url)
5. Commit

**Rollback triggers**:
- Any error during S3 upload
- Any error during DB update
- Context cancellation

### 7.3. S3 Upload vs Transaction

**Problem**: S3 upload không thể rollback nếu DB transaction rollback

**Current Approach**:
- S3 upload trước, update DB sau
- Nếu DB fail → S3 object vẫn tồn tại (orphaned)
- Không có cleanup mechanism cho orphaned S3 objects

**Potential Improvement**: 
- Background job để cleanup orphaned objects
- Two-phase commit pattern (nếu cần strict consistency)

---

## 8. IMAGE PROCESSING ARCHITECTURE

### 8.1. Image Resize Flow

**File**: `internal/service/image_resize.go`

**Algorithm**: Bilinear Interpolation

**Steps**:
1. Decode original image
2. Calculate new dimensions (maintain aspect ratio)
3. Validate dimensions (prevent DoS)
4. Bilinear resize với worker pool
5. Save multiple sizes (thumbnail, medium, large)
6. Optimize each size (nếu optimizer enabled)
7. Return URLs cho tất cả sizes

**Key Features**:
- Aspect ratio preservation
- No upscaling (nếu original nhỏ hơn target)
- Dimension limits (max 10000px) để prevent memory exhaustion
- WordPress-style directory structure (`YYYY/MM/`)

### 8.2. Image Optimization Flow

**File**: `internal/service/image_optimizer.go`

**Strategy**: Multi-quality testing + adaptive compression

**JPEG Optimization**:
1. Strip metadata
2. Reduce color depth (slight quantization)
3. Try 3 quality levels (base-5, base, base+5)
4. Choose smallest file size
5. Progressive encoding

**PNG Optimization**:
1. Strip metadata
2. Try multiple compression levels (Default, BestSpeed, BestCompression)
3. Choose smallest file size

**Color Depth Reduction**:
```go
// * Slight quantization: round to nearest 4 để reduce color variations
c.R = c.R &^ 3  // Clear lowest 2 bits
c.G = c.G &^ 3
c.B = c.B &^ 3
```

**Lợi ích**: Giảm color variations → better compression

### 8.3. Bilinear Interpolation Algorithm

**File**: `internal/service/image_resize.go:274-411`

**Algorithm**:
1. Map destination coordinates → source coordinates (floating point)
2. Get 4 corner pixels (x0,y0), (x1,y0), (x0,y1), (x1,y1)
3. Interpolate horizontally first
4. Interpolate vertically
5. Convert back to uint8

**Code**:
```go
func bilinearInterpolate(c11, c21, c12, c22 color.Color, fx, fy float64) color.Color {
    // Convert colors to RGBA
    r11, g11, b11, a11 := c11.RGBA()
    // ...
    
    // * Interpolate horizontally trước
    r1 := toFloat(r11)*(1-fx) + toFloat(r21)*fx
    // ...
    
    // * Sau đó interpolate vertically
    r := r1*(1-fy) + r2*fy
    // ...
}
```

**Quality**: Better than nearest-neighbor, faster than bicubic

---

## 9. CONFIGURATION MANAGEMENT

### 9.1. Environment-based Configuration

**File**: `internal/config/builder.go:287-431`

**Strategy**:
- Load `.env` file (nếu có)
- Read environment variables
- Apply defaults
- Validate values
- Build config struct

**Features**:
- BOM handling cho `.env` files
- Multiple `.env` file support (`.env`, `.env.local`)
- Type conversion (string → int, duration)
- Range validation (presigned URL expiry, rate limit requests)

### 9.2. Configuration Validation

**Examples**:
```go
// Presigned URL expiry: 1 minute - 7 days
if presignedURLExpiry < 1 {
    presignedURLExpiry = 1
}
if presignedURLExpiry > 60480 { // Max 7 days
    presignedURLExpiry = 60480
}

// WordPress BaseURL validation
if !strings.HasPrefix(wpBaseURL, "http://") && !strings.HasPrefix(wpBaseURL, "https://") {
    return nil, fmt.Errorf("invalid WordPress base URL: must start with http:// or https://")
}
```

### 9.3. Default Values

**Strategy**: Sensible defaults với override capability

**Examples**:
- Port: `8080`
- Max upload size: `2MB` (2<<20)
- Rate limit: `100 requests/minute`
- Concurrency limit: `500`
- JPEG quality: `85`
- Database connections: `25 max open, 5 max idle`

---

## 10. DATABASE SCHEMA & MIGRATIONS

### 10.1. Schema Design

**File**: `internal/database/migrations/001_create_uploads_table.sql`

**Table: `uploads`**
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
```

**Indexes**:
- `idx_uploads_status` (status) - cho queries theo status
- `idx_uploads_created_at` (created_at DESC) - cho sorting
- `idx_uploads_s3_key` (s3_key) - cho lookups

**Design Decisions**:
- `s3_key` và `s3_url` nullable (pending status chưa có)
- `error` TEXT nullable (chỉ có khi failed)
- `status` VARCHAR(20) với constraint (pending, completed, failed)

### 10.2. Migration System

**File**: `cmd/migrate/main.go`

**Features**:
- Version-based migrations (numeric prefix: `001_`, `002_`, ...)
- Transaction support (rollback on failure)
- Migration tracking table (`schema_migrations`)
- Status reporting
- Target version support

**Naming Convention**: `{version}_{description}.sql`

**Safety**:
- SQL identifier quoting (pq.QuoteIdentifier)
- Version validation (numeric only)
- Sort by numeric comparison (not lexicographic)

---

## 11. METRICS & OBSERVABILITY

### 11.1. Metrics Collection

**File**: `internal/metrics/metrics.go`

**Metrics Types**:

1. **Request Metrics**:
   - Count per endpoint
   - Average duration per endpoint
   - Error count per endpoint

2. **Upload Metrics**:
   - Total uploads
   - Success/failed counts
   - Average upload duration

3. **Image Processing Metrics**:
   - Resize count
   - Optimization count
   - Bytes saved

**Storage**: In-memory (global singleton)

**Limits**:
- Max 1000 endpoints tracked
- Last 100 durations per endpoint
- Thread-safe (RWMutex)

### 11.2. Logging Strategy

**Pattern**: Structured logging với request ID

**File**: `internal/middleware/logging.go`

```go
log.Printf(
    "[%s] %s %s %s %v",
    requestID,    // Request ID từ context
    r.Method,     // HTTP method
    r.RequestURI, // URI
    r.RemoteAddr, // Client IP
    time.Since(start), // Duration
)
```

**Log Levels**: 
- Info: Normal operations
- Warning: Non-critical errors (temp file cleanup failures)
- Error: Critical errors (upload failures)

### 11.3. Health Check

**File**: `internal/handlers/api_handler.go:259-315`

**Checks**:
1. Disk space (basic, cross-platform TBD)
2. Database connectivity (ping + connection stats)
3. S3 connectivity (HeadBucket)

**Response Format**:
```json
{
  "status": "ok",
  "service": "s3-upload-api",
  "max_size": 2097152,
  "checks": {
    "disk": {...},
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
```

---

## 12. FILE HANDLING & TEMP FILES

### 12.1. Temp File Strategy

**Pattern**: Create → Use → Cleanup (defer)

**File**: `internal/service/upload_service.go`

```go
tempFile, err := os.CreateTemp(s.uploadDir, "upload-*"+filepath.Ext(filename))
tempFileName := tempFile.Name()

defer func() {
    if removeErr := os.Remove(tempFileName); removeErr != nil {
        log.Printf("[UploadService] Cảnh báo: Không thể xóa file tạm %s: %v", tempFileName, removeErr)
    }
}()
defer tempFile.Close()
```

**Lợi ích**:
- Automatic cleanup (even on panic)
- Unique filenames (pattern matching)
- Proper error handling

### 12.2. File Copy với Timeout

**Challenge**: `io.Copy` không hỗ trợ context cancellation

**Solution**: Goroutine + channel + file close on timeout

```go
done := make(chan error, 1)
var bytesWritten int64

go func() {
    bytesWritten, err = io.Copy(tempFile, limitedReader)
    done <- err
}()

select {
case err := <-done:
    // success
case <-copyCtx.Done():
    // ! Đóng file để interrupt I/O
    tempFile.Close()
    // Wait for goroutine với timeout
    // ...
}
```

### 12.3. File Size Validation

**Triple Validation**:
1. Header size check (before copy)
2. Actual bytes written check (after copy)
3. Truncation detection (maxSize+1 check)

**File**: `internal/service/upload_service.go:91-159`

```go
// ? Dùng maxSize+1 để phát hiện file vượt limit
limitedReader := io.LimitReader(file, maxSize+1)

// After copy:
if bytesWritten > maxSize {
    return nil, NewFileSizeError(...)
}
if bytesWritten == maxSize+1 {
    return nil, NewFileSizeError(...) // Truncated
}
```

---

## 13. DEPENDENCY INJECTION & FACTORIES

### 13.1. Container Initialization

**File**: `internal/container/container.go:49-202`

**Initialization Order**:
1. Load config
2. Create directories
3. Initialize HTTP client
4. Initialize AWS S3 client
5. Initialize database (if enabled)
6. Create repositories (factory)
7. Create services (factory)
8. Create handlers
9. Setup middleware chain

**Error Handling**: Fail-fast (return error nếu bất kỳ step nào fail)

### 13.2. Factory Pattern Usage

**Repository Factory**:
```go
repoFactory := repository.NewRepositoryFactory()
s3Repo, err := repoFactory.CreateRepository(
    repository.RepositoryTypeS3,
    uploader,
    s3Client,
    cfg.AWS.BucketName,
)
```

**Service Factory**:
```go
serviceFactory := service.NewServiceFactory(repoFactory, db)
uploadService, err := serviceFactory.CreateService(
    service.ServiceTypeUpload,
    s3Repo,
    uploadRepo,
    // ... config params
)
```

**Lợi ích**:
- Centralized creation logic
- Easy to extend (thêm repository/service types)
- Dependency injection

---

## 14. MIDDLEWARE ARCHITECTURE

### 14.1. Middleware Chain Order

**File**: `internal/container/container.go:267-297`

**Order** (innermost → outermost):
1. Request ID (innermost - để logging có request ID)
2. Logging
3. Metrics
4. CSRF
5. Rate Limiting
6. Concurrency Limiting (outermost - để limit trước)

**Lý do thứ tự**:
- Request ID phải đầu tiên để tất cả logs có ID
- Metrics sau logging để capture all requests
- Rate/Concurrency limiting ở ngoài để reject sớm

### 14.2. API-specific Middleware

**Applied separately**:
- CORS (chỉ cho API routes)
- API Key Auth (chỉ cho API routes nếu required)

**Implementation**:
```go
apiHandler := http.Handler(apiMux)
if len(c.Config.API.CORSOrigins) > 0 {
    apiHandler = cors.CORSMiddleware(apiHandler)
}
if c.Config.API.RequireAPIKey {
    apiHandler = middleware.APIKeyAuth(...)(apiHandler)
}
mux.Handle("/api/", apiHandler)
```

### 14.3. Middleware Response Wrapping

**Metrics Middleware**:
```go
type responseWriter struct {
    http.ResponseWriter
    statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
}
```

**Purpose**: Capture status code để record errors (>= 400)

---

## 15. GRACEFUL SHUTDOWN

### 15.1. Shutdown Sequence

**File**: `cmd/server/main.go:39-55`

**Steps**:
1. Wait for signal (SIGINT/SIGTERM)
2. Stop rate limiters (cancel goroutines)
3. Close database connections
4. Server shutdown với timeout
5. Wait for server context done

**Implementation**:
```go
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

ctn.Shutdown() // Stop rate limiters, close DB

ctx, cancel := context.WithTimeout(..., ctn.Config.Server.ShutdownTimeout)
defer cancel()

server.Shutdown(ctx) // Graceful shutdown
<-serverCtx.Done()
```

**Timeout**: Configurable (default: 5 seconds)

### 15.2. Resource Cleanup

**Container Shutdown**:
```go
func (c *Container) Shutdown() {
    // Stop all rate limiters to prevent goroutine leaks
    for _, rl := range c.RateLimiters {
        rl.Stop()
    }
    
    // Close database connection if exists
    if c.DB != nil {
        c.DB.Close()
    }
}
```

**Rate Limiter Stop**:
```go
func (rl *RateLimiter) Stop() {
    rl.cancel() // Cancel context → cleanup goroutine exits
}
```

---

## 16. WORDPRESS INTEGRATION

### 16.1. WordPress Upload Flow

**File**: `internal/handlers/wp_handler.go`

**Endpoint**: `POST /api/v1/wp-upload`

**Features**:
- Image-only uploads
- Multiple size generation (thumbnail, medium, large, ...)
- WordPress-style directory structure (`YYYY/MM/`)
- Image optimization
- Original + resized URLs

**Response Format**:
```json
{
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
    }
  ]
}
```

### 16.2. WordPress Directory Structure

**Pattern**: `{wpUploadsDir}/{YYYY}/{MM}/{filename}`

**Example**: `./wp-uploads/2024/01/image-150x150.jpg`

**URL Generation**:
```go
func (s *ImageResizeService) generateURL(year, month, filename string) string {
    base := strings.TrimSuffix(s.baseURL, "/")
    return base + "/wp-content/uploads/" + year + "/" + month + "/" + filename
}
```

### 16.3. Image Size Configuration

**Default WordPress Sizes**:
- thumbnail: 150x150
- medium: 300x300
- medium_large: 768x0 (width only)
- large: 1024x1024

**Configurable via Environment**:
```
WORDPRESS_IMAGE_SIZES="thumbnail:150x150,medium:300x300,large:1024x1024"
```

---

## 17. TESTING INFRASTRUCTURE

### 17.1. Test Files

**Existing Tests**:
- `internal/utils/file_size_test.go`
- `internal/utils/filename_test.go`
- `internal/middleware/request_id_test.go`

### 17.2. Test Coverage Gaps

**Chưa có tests cho**:
- Service layer (upload_service, image_resize, image_optimizer)
- Repository layer (s3_repository, upload_repository)
- Handlers (api_handler, wp_handler)
- Middleware (csrf, ratelimit, concurrency)
- Container initialization

**Recommendation**: 
- Unit tests cho business logic
- Integration tests cho handlers
- Mock repositories cho service tests

---

## 18. CODE QUALITY & BEST PRACTICES

### 18.1. SOLID Principles

**Single Responsibility**:
- ✅ Mỗi package có responsibility rõ ràng
- ✅ Handlers chỉ xử lý HTTP
- ✅ Services chỉ business logic
- ✅ Repositories chỉ data access

**Open/Closed**:
- ✅ Interface-based design (dễ extend)
- ✅ Factory pattern (dễ thêm types mới)

**Liskov Substitution**:
- ✅ Repository interfaces có thể substitute

**Interface Segregation**:
- ✅ Interfaces nhỏ, focused
- ✅ Không có bloated interfaces

**Dependency Inversion**:
- ✅ Depend on abstractions (interfaces)
- ✅ Container injects dependencies

### 18.2. Error Handling Best Practices

- ✅ Error wrapping với `%w`
- ✅ Custom error types
- ✅ Error sanitization ở handler layer
- ✅ Context trong error messages

### 18.3. Concurrency Best Practices

- ✅ Mutex protection cho shared state
- ✅ Context cancellation support
- ✅ Goroutine cleanup (defer, Stop methods)
- ✅ Worker pools với proper synchronization

### 18.4. Security Best Practices

- ✅ Input validation & sanitization
- ✅ SQL injection prevention (prepared statements)
- ✅ CSRF protection
- ✅ Rate limiting
- ✅ Concurrency limiting
- ✅ Constant-time comparisons
- ✅ No credential hardcoding

---

## 19. POTENTIAL IMPROVEMENTS

### 19.1. Testing

**Missing**:
- Unit tests cho services
- Integration tests
- E2E tests
- Mock implementations

### 19.2. Observability

**Missing**:
- Distributed tracing (OpenTelemetry, Jaeger)
- Structured logging (logrus, zap)
- Metrics export (Prometheus)
- Alerting

### 19.3. Performance

**Potential**:
- Redis caching cho metrics
- CDN integration (CloudFront)
- Async upload processing (queue system)
- Connection pooling tuning

### 19.4. Security

**Potential**:
- JWT authentication (thay vì API key đơn giản)
- File virus scanning
- Content Security Policy headers
- Rate limiting per user (thay vì per IP)

### 19.5. Architecture

**Potential**:
- Event-driven architecture (event bus)
- Background job processing (SQS, RabbitMQ)
- Multi-tenant support
- File versioning
- Orphaned S3 object cleanup

---

## 20. TECHNICAL DEBT & LIMITATIONS

### 20.1. Known Limitations

1. **Disk Space Check**: Chưa implement cross-platform (chỉ placeholder)
2. **Orphaned S3 Objects**: Không có cleanup mechanism nếu DB transaction fail
3. **Metrics Storage**: In-memory only (lost on restart)
4. **WebP Support**: Fallback to JPEG (chưa full WebP encoding)
5. **Testing**: Thiếu test coverage

### 20.2. Technical Debt

1. **Error Messages**: Một số error messages tiếng Việt, một số tiếng Anh (inconsistent)
2. **Logging**: Chưa có structured logging, log levels
3. **Config Validation**: Một số validations có thể strict hơn
4. **Documentation**: Thiếu API documentation (OpenAPI/Swagger)

---

## 21. SUMMARY

### 21.1. Strengths

✅ **Clean Architecture**: Rõ ràng, maintainable  
✅ **Security**: Comprehensive security measures  
✅ **Performance**: Optimized với worker pools, connection pooling  
✅ **Error Handling**: Proper error wrapping và sanitization  
✅ **Concurrency**: Safe concurrency patterns  
✅ **Graceful Shutdown**: Proper resource cleanup  
✅ **Configuration**: Flexible, environment-based  
✅ **Code Quality**: SOLID principles, best practices  

### 21.2. Areas for Improvement

⚠️ **Testing**: Thiếu test coverage  
⚠️ **Observability**: Cần structured logging, tracing  
⚠️ **Documentation**: Cần API docs  
⚠️ **Error Consistency**: Inconsistent error message languages  
⚠️ **Orphaned Objects**: Cần cleanup mechanism  

### 21.3. Overall Assessment

**Codebase Quality**: ⭐⭐⭐⭐ (4/5)

- Architecture: Excellent
- Security: Excellent
- Performance: Very Good
- Maintainability: Very Good
- Testing: Needs Improvement
- Observability: Needs Improvement

**Recommendation**: 
- Excellent foundation, ready for production
- Prioritize testing và observability improvements
- Consider async processing cho large-scale deployments

---

**Tài liệu được tạo**: 2024  
**Phân tích bởi**: Code Review System  
**Scope**: Toàn bộ codebase (41 Go files)

