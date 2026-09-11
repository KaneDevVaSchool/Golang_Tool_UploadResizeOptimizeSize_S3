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

// defaultAllowedAdminEmails là danh sách tài khoản Google được phép đăng nhập
// admin khi .env không khai báo ADMIN_ALLOWED_EMAILS. Email ngoài danh sách
// này bị chặn ngay ở callback OAuth, không được tự tạo admin_users.
// Toàn bộ phải viết thường - so khớp bằng strings.ToLower.
var defaultAllowedAdminEmails = []string{
	"khoana@hcm.vaschools.edu.vn",
	"ngocntk@hcm.vaschools.edu.vn",
	"toanbq@vaschools.edu.vn",
	"thaoptp@hcm.vaschools.edu.vn",
	"hiennn@vaschools.edu.vn",
	"hoangbh@vaschools.edu.vn",
	"nhunh@hcm.vaschools.edu.vn",
	"thaontp@vaschools.edu.vn",
	"phongcongnghe@vaschools.edu.vn",
}

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
		BasePath:           b.config.AWS.BasePath,
		Endpoint:           endpoint,
		ForcePathStyle:     forcePathStyle,
		UseACL:             useACL,
		UsePresignedURL:    usePresignedURL,
		PresignedURLExpiry: presignedURLExpiry,
	}
	return b
}

func (b *ConfigBuilder) WithS3BasePath(basePath string) *ConfigBuilder {
	b.config.AWS.BasePath = strings.Trim(strings.TrimSpace(basePath), "/")
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

func (b *ConfigBuilder) WithDirectories(uploadDir string) *ConfigBuilder {
	if uploadDir == "" {
		uploadDir = "./uploads"
	}
	b.config.Directories = DirectoriesConfig{
		UploadDir: uploadDir,
	}
	return b
}

func (b *ConfigBuilder) WithDatabase(enabled bool, driver, dataSource string, maxOpen, maxIdle, maxLifetime int, autoMigrate bool) *ConfigBuilder {
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
		AutoMigrate: autoMigrate,
	}
	return b
}

// WithAuth cấu hình Google OAuth + session admin. Không validate creds ở
// đây - thiếu Client ID/Secret vẫn cho server khởi động bình thường,
// BuildFromEnv() chỉ log cảnh báo để không chặn các tính năng khác.
func (b *ConfigBuilder) WithAuth(clientID, clientSecret, redirectURL, sessionSecret, cookieName string, sessionTTL time.Duration, secureCookie bool, allowedEmailDomains, allowedEmails []string) *ConfigBuilder {
	if cookieName == "" {
		cookieName = "vas_admin_session"
	}
	if sessionTTL == 0 {
		sessionTTL = 168 * time.Hour // 7 ngày
	}
	b.config.Auth = AuthConfig{
		GoogleClientID:      clientID,
		GoogleClientSecret:  clientSecret,
		GoogleRedirectURL:   redirectURL,
		SessionSecret:       sessionSecret,
		SessionCookieName:   cookieName,
		SessionTTL:          sessionTTL,
		SecureCookie:        secureCookie,
		AllowedEmailDomains: allowedEmailDomains,
		AllowedEmails:       allowedEmails,
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

// WithSecurity nối các lựa chọn phòng thủ (proxy tin cậy, chống quét, CSP).
func (b *ConfigBuilder) WithSecurity(sec SecurityConfig) *ConfigBuilder {
	b.config.Security = sec
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


	// Database configuration
	dbEnabled := getEnv("DATABASE_ENABLED", "false") == "true"
	dbDriver := getEnv("DATABASE_DRIVER", "mysql")
	dbDataSource := getEnv("DATABASE_URL", "")
	dbMaxOpen := parseInt(getEnv("DATABASE_MAX_OPEN", "25"), 25)
	dbMaxIdle := parseInt(getEnv("DATABASE_MAX_IDLE", "5"), 5)
	dbMaxLifetime := parseInt(getEnv("DATABASE_MAX_LIFETIME", "300"), 300)
	dbAutoMigrate := getEnv("DATABASE_AUTO_MIGRATE", "true") == "true"

	// Google OAuth + admin session configuration
	googleClientID := getEnv("GOOGLE_CLIENT_ID", "")
	googleClientSecret := getEnv("GOOGLE_CLIENT_SECRET", "")
	googleRedirectURL := getEnv("GOOGLE_REDIRECT_URL", "")
	sessionSecret := getEnv("SESSION_SECRET", "")
	sessionTTLHours := parseInt(getEnv("SESSION_TTL_HOURS", "168"), 168)
	if sessionTTLHours < 1 {
		sessionTTLHours = 168
	}
	var allowedEmailDomains []string
	if raw := getEnv("ADMIN_ALLOWED_EMAIL_DOMAIN", ""); raw != "" {
		for _, d := range strings.Split(raw, ",") {
			d = strings.ToLower(strings.TrimSpace(d))
			if d != "" {
				allowedEmailDomains = append(allowedEmailDomains, d)
			}
		}
	}
	// Whitelist email cụ thể: chỉ đúng các tài khoản này mới đăng nhập admin
	// được, email khác (kể cả cùng domain trường) đều bị từ chối. Override
	// bằng ADMIN_ALLOWED_EMAILS trong .env khi cần đổi danh sách.
	allowedEmails := defaultAllowedAdminEmails
	if raw := getEnv("ADMIN_ALLOWED_EMAILS", ""); raw != "" {
		allowedEmails = nil
		for _, e := range strings.Split(raw, ",") {
			e = strings.ToLower(strings.TrimSpace(e))
			if e != "" {
				allowedEmails = append(allowedEmails, e)
			}
		}
	}
	// Cookie session dùng chung quy ước "secure theo production" với CSRF
	// cookie hiện có - đã tính csrfSecureCookie ở trên nên tái dùng luôn.
	authSecureCookie := csrfSecureCookie

	if googleClientID == "" || googleClientSecret == "" {
		log.Println("⚠ GOOGLE_CLIENT_ID/GOOGLE_CLIENT_SECRET chưa cấu hình - /auth/google/* sẽ trả lỗi 503 cho tới khi được điền vào .env")
	}

	forcePathStyle := getEnv("S3_FORCE_PATH_STYLE", "false") == "true"
	s3Endpoint := getEnv("S3_ENDPOINT", "")

	// --- Bảo mật: proxy tin cậy, chống quét, CSP ---
	//
	// Mặc định tin loopback vì kiến trúc triển khai là Nginx chạy cùng máy
	// proxy sang 127.0.0.1:8080 (xem deploy/nginx/). Người đặt app sau một
	// proxy khác (Cloudflare, load balancer riêng) phải khai dải IP của proxy
	// đó, nếu không rate limit sẽ gom mọi khách vào cùng một bộ đếm.
	trustedProxies := splitAndTrim(getEnv("TRUSTED_PROXIES", "127.0.0.0/8,::1/128"))

	botGuardEnabled := getEnv("BOT_GUARD_ENABLED", "true") == "true"
	botGuardMaxRequests := parseInt(getEnv("BOT_GUARD_MAX_REQUESTS_PER_MINUTE", "240"), 240)
	botGuardMaxPaths := parseInt(getEnv("BOT_GUARD_MAX_PATHS_PER_MINUTE", "150"), 150)
	botGuardBlockMinutes := parseInt(getEnv("BOT_GUARD_BLOCK_MINUTES", "10"), 10)

	// HSTS mặc định theo production, nhưng vẫn cho tắt tường minh: bật HSTS
	// khi site còn phục vụ HTTP sẽ khoá trình duyệt khỏi site suốt max-age.
	enableHSTS := appEnv == "production"
	if v := getEnv("SECURITY_HSTS_ENABLED", ""); v != "" {
		enableHSTS = v == "true"
	}

	// CSP phải biết domain S3, nếu không img-src 'self' sẽ chặn chính ảnh tác
	// phẩm. Tự suy từ cấu hình S3 đang dùng, cộng thêm khai báo tuỳ ý.
	cspImageSources := splitAndTrim(getEnv("SECURITY_CSP_IMAGE_SOURCES", ""))
	cspImageSources = append(cspImageSources, deriveS3Origins(
		getEnv("S3_BUCKET_NAME", ""),
		getEnv("AWS_REGION", "us-east-1"),
		s3Endpoint,
	)...)
	cspConnectSources := splitAndTrim(getEnv("SECURITY_CSP_CONNECT_SOURCES", ""))

	maxJSONBody := int64(parseInt(getEnv("MAX_JSON_BODY_KB", "1024"), 1024)) << 10

	securityCfg := SecurityConfig{
		TrustedProxies:       trustedProxies,
		BotGuardEnabled:      botGuardEnabled,
		BotGuardMaxRequests:  botGuardMaxRequests,
		BotGuardMaxPaths:     botGuardMaxPaths,
		BotGuardBlockMinutes: botGuardBlockMinutes,
		EnableHSTS:           enableHSTS,
		CSPImageSources:      cspImageSources,
		CSPConnectSources:    cspConnectSources,
		MaxJSONBodyBytes:     maxJSONBody,
	}

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
		WithS3BasePath(getEnv("S3_BASE_PATH", "")).
		WithUpload(
			int64(parseInt(getEnv("UPLOAD_MAX_SIZE_MB", "20"), 20))<<20,
			int64(parseInt(getEnv("UPLOAD_ABSOLUTE_MAX_MB", "200"), 200))<<20,
			uploadTimeout,
		).
		WithDatabase(dbEnabled, dbDriver, dbDataSource, dbMaxOpen, dbMaxIdle, dbMaxLifetime, dbAutoMigrate).
		WithDirectories("./uploads").
		WithRateLimit(rateLimitEnabled, rateLimitRequests, rateLimitWindow, rateLimitCleanup).
		WithSecurity(securityCfg).
		WithCSRF(csrfEnabled, csrfSecureCookie).
		WithConcurrency(concurrencyEnabled, maxConcurrent, acquireTimeout).
		WithAPI(apiEnabled, apiKey, corsOrigins, requireAPIKey).
		WithAuth(googleClientID, googleClientSecret, googleRedirectURL, sessionSecret, "", time.Duration(sessionTTLHours)*time.Hour, authSecureCookie, allowedEmailDomains, allowedEmails).
		Build()
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// splitAndTrim tách danh sách ngăn cách bằng dấu phẩy, bỏ phần tử rỗng.
func splitAndTrim(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// deriveS3Origins suy ra origin phục vụ ảnh từ cấu hình S3 đang dùng.
//
// Không bắt người vận hành khai lại domain S3 trong một biến CSP riêng: họ đã
// khai bucket/region/endpoint rồi, và quên đồng bộ hai chỗ sẽ khiến CSP chặn
// đúng ảnh tác phẩm - lỗi chỉ lộ ra trên trình duyệt người dùng cuối, không
// thấy trong log server.
func deriveS3Origins(bucket, region, endpoint string) []string {
	var origins []string

	if e := strings.TrimSpace(endpoint); e != "" {
		// Endpoint tuỳ chỉnh (MinIO/LocalStack) đã là URL đầy đủ.
		origins = append(origins, strings.TrimSuffix(e, "/"))
		return origins
	}

	b := strings.TrimSpace(bucket)
	if b == "" {
		return origins
	}
	r := strings.TrimSpace(region)
	if r == "" {
		r = "us-east-1"
	}

	// Hai dạng URL S3 đều có thể xuất hiện tuỳ cách sinh link
	// (virtual-hosted style và dạng kèm region), khai cả hai cho chắc.
	origins = append(origins,
		"https://"+b+".s3."+r+".amazonaws.com",
		"https://"+b+".s3.amazonaws.com",
	)
	return origins
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
