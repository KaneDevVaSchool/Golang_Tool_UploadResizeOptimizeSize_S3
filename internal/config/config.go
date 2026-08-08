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
	API         APIConfig
	WordPress   WordPressConfig
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
	UseACL             bool
	UsePresignedURL    bool
	PresignedURLExpiry int // minutes
}

type UploadConfig struct {
	MaxSize         int64         // max size per request / chunk (default 20MB)
	AbsoluteMaxSize int64         // max total file size via chunked upload
	UploadTimeout   time.Duration // timeout for S3 upload operations
}

type DirectoriesConfig struct {
	UploadDir    string
	WPUploadsDir string // WordPress uploads directory
}

type WordPressConfig struct {
	Enabled      bool
	BaseURL      string // Base URL for WordPress site (e.g., https://example.com)
	ImageSizes   []ImageSizeConfig
	Optimization ImageOptimizationConfig
}

type ImageOptimizationConfig struct {
	Enabled     bool
	JPEGQuality int // 0-100, 0 means auto
	PNGQuality  int // 0-100, 0 means auto
	EnableWebP  bool
}

type ImageSizeConfig struct {
	Name   string
	Width  int
	Height int
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
	MaxLifetime int // seconds
}

func Load() (*Config, error) {
	builder := NewConfigBuilder()
	return builder.BuildFromEnv()
}
