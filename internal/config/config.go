package config

import "time"

type Config struct {
	Server      ServerConfig
	AWS         AWSConfig
	Upload      UploadConfig
	Database    DatabaseConfig
	Directories DirectoriesConfig
	RateLimit   RateLimitConfig
	CSRF        CSRFConfig
	Concurrency ConcurrencyConfig
	Security    SecurityConfig
	API         APIConfig
	Auth        AuthConfig
}

type ServerConfig struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration // Timeout for graceful server shutdown
}

type RateLimitConfig struct {
	Enabled         bool
	Requests        int           // requests per window
	Window          time.Duration // time window
	CleanupInterval time.Duration
}

// SecurityConfig gom các lựa chọn phòng thủ không thuộc rate limit.
type SecurityConfig struct {
	// TrustedProxies là dải CIDR được phép đặt X-Forwarded-For/X-Real-IP.
	// Rỗng nghĩa là không tin proxy nào (dùng thẳng RemoteAddr).
	TrustedProxies []string
	// BotGuardEnabled bật lớp nhận diện công cụ tải site hàng loạt.
	BotGuardEnabled bool
	// BotGuardMaxRequests / BotGuardMaxPaths là ngưỡng nhịp bị coi là máy quét.
	BotGuardMaxRequests int
	BotGuardMaxPaths    int
	// BotGuardBlockMinutes là thời gian giữ hình phạt sau khi vượt ngưỡng.
	BotGuardBlockMinutes int
	// EnableHSTS chỉ bật khi site đã chạy HTTPS hoàn toàn.
	EnableHSTS bool
	// CSPImageSources / CSPConnectSources khai báo origin ngoài (S3/CDN) được
	// phép tải ảnh và gọi API - thiếu thì CSP chặn chính ảnh tác phẩm.
	CSPImageSources   []string
	CSPConnectSources []string
	// MaxJSONBodyBytes là trần body cho endpoint không phải upload.
	MaxJSONBodyBytes int64
}

type ConcurrencyConfig struct {
	Enabled        bool
	MaxConcurrent  int64         // maximum concurrent uploads
	AcquireTimeout time.Duration // timeout for acquiring semaphore
}

type CSRFConfig struct {
	Enabled      bool
	SecureCookie bool
}

type AWSConfig struct {
	// Note: AccessKeyID and SecretAccessKey removed for security
	// AWS SDK will use credential chain: environment variables, IAM roles, credentials file, EC2 metadata
	Region             string
	BucketName         string
	BasePath           string // optional key prefix, e.g. vaschools-uploads
	Endpoint           string // optional custom endpoint (MinIO / LocalStack)
	ForcePathStyle     bool
	UseACL             bool
	UsePresignedURL    bool
	PresignedURLExpiry int // minutes
}

type UploadConfig struct {
	// MaxSize là trần mặc định cho một request upload (mặc định 20MB).
	MaxSize int64
	// AbsoluteMaxSize là trần cứng cho một file, dùng cho đường upload ảnh
	// tác phẩm vốn cho phép file lớn hơn MaxSize.
	AbsoluteMaxSize int64
	UploadTimeout   time.Duration // timeout cho thao tác upload lên S3
}

type DirectoriesConfig struct {
	UploadDir string
}

type APIConfig struct {
	Enabled       bool
	APIKey        string
	CORSOrigins   []string
	RequireAPIKey bool
}

type DatabaseConfig struct {
	Enabled     bool
	Driver      string
	DataSource  string
	MaxOpen     int
	MaxIdle     int
	MaxLifetime int  // seconds
	AutoMigrate bool // chạy internal/database/migrations/*.sql tự động lúc khởi động
}

// AuthConfig cấu hình đăng nhập admin qua Google OAuth + session cookie.
// GoogleClientID/Secret có thể rỗng lúc khởi động (server vẫn chạy được,
// chỉ /auth/google/* trả 503) - user tạo Google Cloud Console credentials sau.
type AuthConfig struct {
	GoogleClientID      string
	GoogleClientSecret  string
	GoogleRedirectURL   string
	SessionSecret       string
	SessionCookieName   string
	SessionTTL          time.Duration
	SecureCookie        bool     // theo APP_ENV=production, giống CSRFConfig.SecureCookie
	AllowedEmailDomains []string // optional, vd ["vaschools.edu.vn","hcm.vaschools.edu.vn"] - rỗng nghĩa là không giới hạn
	// AllowedEmails là whitelist email chính xác được phép đăng nhập admin.
	// Khi danh sách này không rỗng, nó là điều kiện quyết định: chỉ đúng các
	// email trong danh sách mới vào được (AllowedEmailDomains bị bỏ qua).
	AllowedEmails []string
}

func Load() (*Config, error) {
	builder := NewConfigBuilder()
	return builder.BuildFromEnv()
}
