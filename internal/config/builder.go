package config

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

var loadEnvOnce sync.Once

type ConfigBuilder struct {
	config *Config
}

func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config: &Config{},
	}
}

func (b *ConfigBuilder) WithServer(port string, readTimeout, writeTimeout, idleTimeout, shutdownTimeout time.Duration) *ConfigBuilder {
	if port == "" {
		port = "8080"
	}
	if readTimeout == 0 {
		readTimeout = 15 * time.Second
	}
	if writeTimeout == 0 {
		writeTimeout = 30 * time.Second // Increased for uploads
	}
	if idleTimeout == 0 {
		idleTimeout = 60 * time.Second
	}
	if shutdownTimeout == 0 {
		shutdownTimeout = 5 * time.Second // Default 5 seconds for graceful shutdown
	}
	b.config.Server = ServerConfig{
		Port:            port,
		ReadTimeout:     readTimeout,
		WriteTimeout:    writeTimeout,
		IdleTimeout:     idleTimeout,
		ShutdownTimeout: shutdownTimeout,
	}
	return b
}

func (b *ConfigBuilder) WithAWS(region, bucketName string, useACL bool, usePresignedURL bool, presignedURLExpiry int) *ConfigBuilder {
	return b.WithAWSFull(region, bucketName, "", false, useACL, usePresignedURL, presignedURLExpiry)
}

func (b *ConfigBuilder) WithAWSFull(region, bucketName, endpoint string, forcePathStyle, useACL, usePresignedURL bool, presignedURLExpiry int) *ConfigBuilder {
	if region == "" {
		region = "us-east-1"
	}
	if presignedURLExpiry == 0 {
		presignedURLExpiry = 60
	}
	b.config.AWS = AWSConfig{
		// Credentials removed - AWS SDK uses credential chain
		Region:             region,
		BucketName:         bucketName,
		Endpoint:           endpoint,
		ForcePathStyle:     forcePathStyle,
		UseACL:             useACL,
		UsePresignedURL:    usePresignedURL,
		PresignedURLExpiry: presignedURLExpiry,
	}
	return b
}

func (b *ConfigBuilder) WithUpload(maxSize, absoluteMaxSize int64, uploadTimeout time.Duration) *ConfigBuilder {
	if maxSize <= 0 {
		maxSize = 20 << 20
	}
	if absoluteMaxSize <= 0 {
		absoluteMaxSize = 200 << 20
	}
	if absoluteMaxSize < maxSize {
		absoluteMaxSize = maxSize
	}
	// Keep chunk count bounded (aligned with maxChunksPerUpload in chunk service)
	const maxChunks = 64
	if absoluteMaxSize/maxSize > maxChunks {
		absoluteMaxSize = maxSize * maxChunks
	}
	if uploadTimeout == 0 {
		uploadTimeout = 5 * time.Minute
	}
	b.config.Upload = UploadConfig{
		MaxSize:         maxSize,
		AbsoluteMaxSize: absoluteMaxSize,
		UploadTimeout:   uploadTimeout,
	}
	return b
}

func (b *ConfigBuilder) WithDirectories(uploadDir, wpUploadsDir string) *ConfigBuilder {
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	if wpUploadsDir == "" {
		wpUploadsDir = "./wp-uploads"
	}
	b.config.Directories = DirectoriesConfig{
		UploadDir:    uploadDir,
		WPUploadsDir: wpUploadsDir,
	}
	return b
}

func (b *ConfigBuilder) WithDatabase(enabled bool, driver, dataSource string, maxOpen, maxIdle, maxLifetime int) *ConfigBuilder {
	if maxOpen == 0 {
		maxOpen = 25
	}
	if maxIdle == 0 {
		maxIdle = 5
	}
	if maxLifetime == 0 {
		maxLifetime = 300 // 5 minutes
	}
	b.config.Database = DatabaseConfig{
		Enabled:     enabled,
		Driver:      driver,
		DataSource:  dataSource,
		MaxOpen:     maxOpen,
		MaxIdle:     maxIdle,
		MaxLifetime: maxLifetime,
	}
	return b
}

func (b *ConfigBuilder) WithWordPress(enabled bool, baseURL string, imageSizes []ImageSizeConfig, optimization ImageOptimizationConfig) *ConfigBuilder {
	if len(imageSizes) == 0 {
		// Default WordPress image sizes
		imageSizes = []ImageSizeConfig{
			{Name: "thumbnail", Width: 150, Height: 150},
			{Name: "medium", Width: 300, Height: 300},
			{Name: "medium_large", Width: 768, Height: 0}, // 0 means no limit
			{Name: "large", Width: 1024, Height: 1024},
		}
	}

	// Validate image sizes
	seenNames := make(map[string]bool)
	const maxDimension = 10000
	for i := range imageSizes {
		size := &imageSizes[i]

		// Validate name
		if size.Name == "" {
			size.Name = fmt.Sprintf("size_%d", i)
		}
		if seenNames[size.Name] {
			// Duplicate name, skip this size
			continue
		}
		seenNames[size.Name] = true

		// Validate dimensions
		if size.Width < 0 || size.Height < 0 {
			// Invalid dimensions, use default
			if size.Width < 0 {
				size.Width = 0
			}
			if size.Height < 0 {
				size.Height = 0
			}
		}
		if size.Width > maxDimension {
			size.Width = maxDimension
		}
		if size.Height > maxDimension {
			size.Height = maxDimension
		}
	}

	b.config.WordPress = WordPressConfig{
		Enabled:      enabled,
		BaseURL:      baseURL,
		ImageSizes:   imageSizes,
		Optimization: optimization,
	}
	return b
}

func (b *ConfigBuilder) WithRateLimit(enabled bool, requests int, windowMinutes int, cleanupMinutes int) *ConfigBuilder {
	if requests == 0 {
		requests = 100 // Default: 100 requests per window
	}
	if windowMinutes == 0 {
		windowMinutes = 1 // Default: 1 minute window
	}
	if cleanupMinutes == 0 {
		cleanupMinutes = 10 // Default: cleanup every 10 minutes
	}
	b.config.RateLimit = RateLimitConfig{
		Enabled:         enabled,
		Requests:        requests,
		Window:          time.Duration(windowMinutes) * time.Minute,
		CleanupInterval: time.Duration(cleanupMinutes) * time.Minute,
	}
	return b
}

func (b *ConfigBuilder) WithCSRF(enabled bool, secureCookie bool) *ConfigBuilder {
	b.config.CSRF = CSRFConfig{
		Enabled:      enabled,
		SecureCookie: secureCookie,
	}
	return b
}

func (b *ConfigBuilder) WithConcurrency(enabled bool, maxConcurrent int64, acquireTimeout time.Duration) *ConfigBuilder {
	if maxConcurrent == 0 {
		maxConcurrent = 500 // Default: allow 500 concurrent uploads
	}
	if acquireTimeout == 0 {
		acquireTimeout = 30 * time.Second // Default: 30 seconds timeout for acquiring semaphore
	}
	b.config.Concurrency = ConcurrencyConfig{
		Enabled:        enabled,
		MaxConcurrent:  maxConcurrent,
		AcquireTimeout: acquireTimeout,
	}
	return b
}

func (b *ConfigBuilder) WithAPI(enabled bool, apiKey string, corsOrigins []string, requireAPIKey bool) *ConfigBuilder {
	b.config.API = APIConfig{
		Enabled:       enabled,
		APIKey:        apiKey,
		CORSOrigins:   corsOrigins,
		RequireAPIKey: requireAPIKey,
	}
	return b
}

func (b *ConfigBuilder) Build() (*Config, error) {
	if b.config.AWS.BucketName == "" {
		return nil, ErrMissingBucketName
	}
	if b.config.API.RequireAPIKey && strings.TrimSpace(b.config.API.APIKey) == "" {
		return nil, fmt.Errorf("API_KEY is required when API_REQUIRE_KEY=true")
	}
	return b.config, nil
}

func LoadEnvFile() error {
	var loadErr error
	loadEnvOnce.Do(func() {
		envFiles := []string{".env", ".env.local"}

		for _, envFile := range envFiles {
			if _, err := os.Stat(envFile); err == nil {
				if err := loadEnvFileWithoutBOM(envFile); err != nil {
					log.Printf("Lỗi: Không thể load file %s", envFile)
					log.Printf("Chi tiết: %v", err)
					log.Printf("Vui lòng kiểm tra format file .env")
					log.Printf("Format đúng: KEY=value (không có $env: hoặc quotes)")
					log.Printf("Ví dụ: AWS_ACCESS_KEY_ID=your_key")
					log.Printf("Xem file .env.example để biết format đúng")
					log.Printf("Hoặc chạy: .\fix-env.ps1 để tự động sửa")
					loadErr = err
					continue
				}
				log.Printf("✓ Đã load file %s thành công", envFile)
				loadErr = nil
				return
			}
		}

		log.Println("Không tìm thấy file .env, sử dụng environment variables")
	})
	return loadErr
}

func loadEnvFileWithoutBOM(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}

	tempFile, err := os.CreateTemp("", "env-*.tmp")
	if err != nil {
		return err
	}
	tempFileName := tempFile.Name()

	// Ensure cleanup in all cases
	defer func() {
		if err := os.Remove(tempFileName); err != nil {
			log.Printf("Warning: Failed to remove temp file %s: %v", tempFileName, err)
		}
	}()
	defer tempFile.Close()

	if _, err := tempFile.Write(data); err != nil {
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}

	return godotenv.Load(tempFileName)
}

func (b *ConfigBuilder) BuildFromEnv() (*Config, error) {
	LoadEnvFile()

	useACL := getEnv("S3_USE_ACL", "false") == "true"
	usePresignedURL := getEnv("S3_USE_PRESIGNED_URL", "false") == "true"
	presignedURLExpiry := parseInt(getEnv("S3_PRESIGNED_URL_EXPIRY", "60"), 60)
	if presignedURLExpiry < 1 {
		presignedURLExpiry = 1
	}
	if presignedURLExpiry > 60480 { // Max 7 days
		presignedURLExpiry = 60480
	}

	rateLimitEnabled := getEnv("RATE_LIMIT_ENABLED", "true") == "true"
	rateLimitRequests := parseInt(getEnv("RATE_LIMIT_REQUESTS", "100"), 100)
	if rateLimitRequests < 1 {
		rateLimitRequests = 1
	}
	if rateLimitRequests > 10000 {
		rateLimitRequests = 10000
	}
	rateLimitWindow := parseInt(getEnv("RATE_LIMIT_WINDOW_MINUTES", "1"), 1)
	if rateLimitWindow < 1 {
		rateLimitWindow = 1
	}
	rateLimitCleanup := parseInt(getEnv("RATE_LIMIT_CLEANUP_MINUTES", "10"), 10)
	if rateLimitCleanup < 1 {
		rateLimitCleanup = 1
	}

	csrfEnabled := getEnv("CSRF_ENABLED", "true") == "true"
	csrfSecureCookie := getEnv("CSRF_SECURE_COOKIE", "false") == "true"

	// Server timeouts (raised for large / chunked uploads)
	readTimeout := time.Duration(parseInt(getEnv("SERVER_READ_TIMEOUT_SECONDS", "180"), 180)) * time.Second
	writeTimeout := time.Duration(parseInt(getEnv("SERVER_WRITE_TIMEOUT_SECONDS", "180"), 180)) * time.Second
	idleTimeout := time.Duration(parseInt(getEnv("SERVER_IDLE_TIMEOUT_SECONDS", "120"), 120)) * time.Second
	shutdownTimeout := time.Duration(parseInt(getEnv("SERVER_SHUTDOWN_TIMEOUT_SECONDS", "30"), 30)) * time.Second

	// Upload timeout (S3 put / multipart)
	uploadTimeout := time.Duration(parseInt(getEnv("UPLOAD_TIMEOUT_SECONDS", "300"), 300)) * time.Second

	// Concurrency limit
	concurrencyEnabled := getEnv("CONCURRENCY_LIMIT_ENABLED", "true") == "true"
	maxConcurrent := int64(parseInt(getEnv("MAX_CONCURRENT_UPLOADS", "500"), 500))
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	if maxConcurrent > 10000 {
		maxConcurrent = 10000 // Reasonable upper limit
	}
	acquireTimeout := time.Duration(parseInt(getEnv("CONCURRENCY_ACQUIRE_TIMEOUT_SECONDS", "30"), 30)) * time.Second

	// API configuration
	appEnv := strings.ToLower(strings.TrimSpace(getEnv("APP_ENV", "development")))
	apiEnabled := getEnv("API_ENABLED", "true") == "true"
	apiKey := getEnv("API_KEY", "")
	requireAPIKey := getEnv("API_REQUIRE_KEY", "false") == "true"
	corsOriginsStr := getEnv("CORS_ORIGINS", "*")
	var corsOrigins []string
	if corsOriginsStr != "" {
		corsOrigins = strings.Split(corsOriginsStr, ",")
		for i := range corsOrigins {
			corsOrigins[i] = strings.TrimSpace(corsOrigins[i])
		}
	}

	if appEnv == "production" {
		if !requireAPIKey || strings.TrimSpace(apiKey) == "" {
			return nil, fmt.Errorf("production requires API_REQUIRE_KEY=true and a non-empty API_KEY")
		}
		if len(corsOrigins) == 0 || (len(corsOrigins) == 1 && corsOrigins[0] == "*") {
			return nil, fmt.Errorf("production requires an explicit CORS_ORIGINS allowlist (not *)")
		}
		if getEnv("CSRF_SECURE_COOKIE", "") == "" {
			csrfSecureCookie = true
		}
	}

	// WordPress configuration
	wpEnabled := getEnv("WORDPRESS_ENABLED", "true") == "true"
	wpBaseURL := getEnv("WORDPRESS_BASE_URL", "http://localhost")
	// Validate BaseURL format
	if wpBaseURL != "" {
		if !strings.HasPrefix(wpBaseURL, "http://") && !strings.HasPrefix(wpBaseURL, "https://") {
			return nil, fmt.Errorf("invalid WordPress base URL: must start with http:// or https://")
		}
	}
	wpUploadsDir := getEnv("WORDPRESS_UPLOADS_DIR", "./wp-uploads")

	// Parse image sizes from env (format: "thumbnail:150x150,medium:300x300,large:1024x1024")
	var wpImageSizes []ImageSizeConfig
	wpSizesStr := getEnv("WORDPRESS_IMAGE_SIZES", "")
	if wpSizesStr != "" {
		sizeConfigs := strings.Split(wpSizesStr, ",")
		for _, sizeConfig := range sizeConfigs {
			parts := strings.Split(strings.TrimSpace(sizeConfig), ":")
			if len(parts) == 2 {
				name := parts[0]
				dimensions := strings.Split(parts[1], "x")
				if len(dimensions) == 2 {
					width := parseInt(dimensions[0], 0)
					height := parseInt(dimensions[1], 0)
					wpImageSizes = append(wpImageSizes, ImageSizeConfig{
						Name:   name,
						Width:  width,
						Height: height,
					})
				}
			}
		}
	}

	// Image optimization configuration
	optEnabled := getEnv("IMAGE_OPTIMIZATION_ENABLED", "true") == "true"
	jpegQuality := parseInt(getEnv("IMAGE_JPEG_QUALITY", "85"), 85)
	if jpegQuality < 0 || jpegQuality > 100 {
		jpegQuality = 85
	}
	pngQuality := parseInt(getEnv("IMAGE_PNG_QUALITY", "90"), 90)
	if pngQuality < 0 || pngQuality > 100 {
		pngQuality = 90
	}
	enableWebP := getEnv("IMAGE_ENABLE_WEBP", "false") == "true"

	wpOptimization := ImageOptimizationConfig{
		Enabled:     optEnabled,
		JPEGQuality: jpegQuality,
		PNGQuality:  pngQuality,
		EnableWebP:  enableWebP,
	}

	// Database configuration
	dbEnabled := getEnv("DATABASE_ENABLED", "false") == "true"
	dbDriver := getEnv("DATABASE_DRIVER", "postgres")
	dbDataSource := getEnv("DATABASE_URL", "")
	dbMaxOpen := parseInt(getEnv("DATABASE_MAX_OPEN", "25"), 25)
	dbMaxIdle := parseInt(getEnv("DATABASE_MAX_IDLE", "5"), 5)
	dbMaxLifetime := parseInt(getEnv("DATABASE_MAX_LIFETIME", "300"), 300)

	forcePathStyle := getEnv("S3_FORCE_PATH_STYLE", "false") == "true"
	s3Endpoint := getEnv("S3_ENDPOINT", "")

	return NewConfigBuilder().
		WithServer(getEnv("PORT", "8080"), readTimeout, writeTimeout, idleTimeout, shutdownTimeout).
		WithAWSFull(
			getEnv("AWS_REGION", "us-east-1"),
			getEnv("S3_BUCKET_NAME", ""),
			s3Endpoint,
			forcePathStyle,
			useACL,
			usePresignedURL,
			presignedURLExpiry,
		).
		WithUpload(
			int64(parseInt(getEnv("UPLOAD_MAX_SIZE_MB", "20"), 20))<<20,
			int64(parseInt(getEnv("UPLOAD_ABSOLUTE_MAX_MB", "200"), 200))<<20,
			uploadTimeout,
		).
		WithDatabase(dbEnabled, dbDriver, dbDataSource, dbMaxOpen, dbMaxIdle, dbMaxLifetime).
		WithDirectories("./uploads", wpUploadsDir).
		WithRateLimit(rateLimitEnabled, rateLimitRequests, rateLimitWindow, rateLimitCleanup).
		WithCSRF(csrfEnabled, csrfSecureCookie).
		WithConcurrency(concurrencyEnabled, maxConcurrent, acquireTimeout).
		WithAPI(apiEnabled, apiKey, corsOrigins, requireAPIKey).
		WithWordPress(wpEnabled, wpBaseURL, wpImageSizes, wpOptimization).
		Build()
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func parseInt(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	if err != nil {
		return defaultValue
	}
	return result
}
