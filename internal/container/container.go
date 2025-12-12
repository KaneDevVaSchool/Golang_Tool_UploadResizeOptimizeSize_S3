package container

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"s3-upload-tool/internal/config"
	"s3-upload-tool/internal/database"
	"s3-upload-tool/internal/handlers"
	"s3-upload-tool/internal/metrics"
	"s3-upload-tool/internal/middleware"
	"s3-upload-tool/internal/repository"
	"s3-upload-tool/internal/service"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

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

// GetDBStats trả về thống kê connection pool nếu DB có sẵn
func (c *Container) GetDBStats() interface{} {
	if c.DB == nil {
		return nil
	}
	stats := c.DB.GetStats()
	return stats
}

func NewContainer() (*Container, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(cfg.Directories.UploadDir, 0755); err != nil {
		return nil, err
	}

	if cfg.WordPress.Enabled {
		if err := os.MkdirAll(cfg.Directories.WPUploadsDir, 0755); err != nil {
			return nil, err
		}
	}

	// * Cấu hình HTTP client cho high concurrency
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

	// ! Timeout 30s để tránh hang khi khởi tạo AWS
	initCtx, initCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer initCancel()

	// * AWS credentials được load theo thứ tự:
	// 1. Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
	// 2. IAM role (nếu chạy trên EC2/ECS/Lambda)
	// 3. AWS credentials file (~/.aws/credentials)
	// 4. EC2 Instance Metadata Service
	awsCfg, err := awsconfig.LoadDefaultConfig(initCtx,
		awsconfig.WithRegion(cfg.AWS.Region),
		awsconfig.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = false
	})

	// * Cấu hình uploader cho concurrent uploads
	uploader := manager.NewUploader(s3Client, func(u *manager.Uploader) {
		u.PartSize = 10 * 1024 * 1024
		u.Concurrency = 5
		u.LeavePartsOnError = false
	})

	// Initialize database if enabled
	var db *database.DB
	var uploadRepo repository.UploadRepository
	if cfg.Database.Enabled && cfg.Database.DataSource != "" {
		dbConfig := database.Config{
			Driver:      cfg.Database.Driver,
			DataSource:  cfg.Database.DataSource,
			MaxOpen:     cfg.Database.MaxOpen,
			MaxIdle:     cfg.Database.MaxIdle,
			MaxLifetime: time.Duration(cfg.Database.MaxLifetime) * time.Second,
		}
		db, err = database.NewDB(dbConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize database: %w", err)
		}
		log.Println("[Container] Database initialized successfully")
		uploadRepo, err = repository.NewUploadRepository(db)
		if err != nil {
			return nil, fmt.Errorf("failed to create upload repository: %w", err)
		}
	}

	repoFactory := repository.NewRepositoryFactory()
	s3Repo, err := repoFactory.CreateRepository(
		repository.RepositoryTypeS3,
		uploader,
		s3Client,
		cfg.AWS.BucketName,
	)
	if err != nil {
		return nil, err
	}

	serviceFactory := service.NewServiceFactory(repoFactory, db)
	uploadService, err := serviceFactory.CreateService(
		service.ServiceTypeUpload,
		s3Repo,
		uploadRepo,
		cfg.AWS.BucketName,
		cfg.AWS.Region,
		cfg.Directories.UploadDir,
		cfg.Upload.MaxSize,
		cfg.Upload.UploadTimeout,
		cfg.AWS.UseACL,
		cfg.AWS.UsePresignedURL,
		cfg.AWS.PresignedURLExpiry,
	)
	if err != nil {
		return nil, err
	}

	apiHandler := handlers.NewAPIHandler(uploadService, s3Repo, cfg.Upload.MaxSize, db)

	// WordPress handler with image resize and optimization
	var wpHandler *handlers.WPHandler
	if cfg.WordPress.Enabled {
		// Convert config image sizes to service image sizes
		imageSizes := make([]service.ImageSizeConfig, len(cfg.WordPress.ImageSizes))
		for i, size := range cfg.WordPress.ImageSizes {
			imageSizes[i] = service.ImageSizeConfig{
				Name:   size.Name,
				Width:  size.Width,
				Height: size.Height,
			}
		}

		// Create image optimizer if enabled
		var optimizer *service.ImageOptimizer
		if cfg.WordPress.Optimization.Enabled {
			optimizer = service.NewImageOptimizer(
				cfg.WordPress.Optimization.JPEGQuality,
				cfg.WordPress.Optimization.PNGQuality,
				cfg.WordPress.Optimization.EnableWebP,
			)
		}

		imageResizeService := service.NewImageResizeService(
			cfg.Directories.WPUploadsDir,
			cfg.WordPress.BaseURL,
			imageSizes,
			optimizer,
		)
		wpHandler = handlers.NewWPHandler(imageResizeService, cfg.Upload.MaxSize, cfg.Directories.UploadDir)
	}

	return &Container{
		Config:           cfg,
		DB:               db,
		S3Client:         s3Client,
		Uploader:         uploader,
		S3Repository:     s3Repo,
		UploadRepository: uploadRepo,
		UploadService:    uploadService,
		APIHandler:       apiHandler,
		WPHandler:        wpHandler,
		RepoFactory:      repoFactory,
		ServiceFactory:   serviceFactory,
	}, nil
}

func (c *Container) GetServerHandler() http.Handler {
	mux := http.NewServeMux()

	// API routes only (API is always enabled for API-only service)
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/api/v1/upload", c.APIHandler.HandleUpload)
	apiMux.HandleFunc("/api/v1/upload-transaction", c.APIHandler.HandleUploadWithTransaction)
	apiMux.HandleFunc("/api/v1/health", c.APIHandler.HandleHealth)

	// Metrics endpoint with rate limiting (if enabled) to prevent abuse
	metricsHandler := handlers.NewMetricsHandler()
	var metricsRateLimiter *middleware.RateLimiter
	if c.Config.RateLimit.Enabled {
		// Create a separate rate limiter for metrics endpoint (stricter limits)
		metricsRateLimiter = middleware.NewRateLimiter(
			10, // Only 10 requests per window for metrics
			c.Config.RateLimit.Window,
			c.Config.RateLimit.CleanupInterval,
		)
		c.RateLimiters = append(c.RateLimiters, metricsRateLimiter)
		apiMux.Handle("/api/v1/metrics", middleware.RateLimitMiddleware(metricsRateLimiter)(http.HandlerFunc(metricsHandler.HandleMetrics)))
	} else {
		apiMux.HandleFunc("/api/v1/metrics", metricsHandler.HandleMetrics)
	}

	// WordPress upload endpoint (with resize)
	if c.Config.WordPress.Enabled && c.WPHandler != nil {
		apiMux.HandleFunc("/api/v1/wp-upload", c.WPHandler.HandleWPUpload)
	}

	// Apply API-specific middleware
	apiHandler := http.Handler(apiMux)

	// CORS middleware
	if len(c.Config.API.CORSOrigins) > 0 {
		cors := middleware.NewCORS(c.Config.API.CORSOrigins)
		apiHandler = cors.CORSMiddleware(apiHandler)
	}

	// API Key authentication (if required)
	if c.Config.API.RequireAPIKey && c.Config.API.APIKey != "" {
		apiHandler = middleware.APIKeyAuth(c.Config.API.APIKey)(apiHandler)
	}

	// Mount API routes
	mux.Handle("/api/", apiHandler)

	// Root endpoint - return API info
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"service":"s3-upload-api","version":"1.0","endpoints":["/api/v1/upload","/api/v1/upload-transaction","/api/v1/health","/api/v1/metrics","/api/v1/wp-upload"]}`))
	})

	// Serve WordPress uploads directory (with security)
	if c.Config.WordPress.Enabled {
		wpFs := secureFileServer(http.Dir(c.Config.Directories.WPUploadsDir))
		mux.Handle("/wp-content/uploads/", http.StripPrefix("/wp-content/uploads/", wpFs))
	}

	handler := http.Handler(mux)

	// Apply middleware in order (last applied is outermost)
	// Request ID middleware should be first to ensure all logs have request ID
	handler = middleware.RequestIDMiddleware(handler)
	handler = middleware.LoggingMiddleware(handler)

	// Metrics middleware (after logging to capture all requests)
	handler = metrics.MetricsMiddleware(handler)

	// Apply CSRF protection if enabled
	if c.Config.CSRF.Enabled {
		csrfProtection := middleware.NewCSRFProtection(c.Config.CSRF.SecureCookie)
		handler = csrfProtection.CSRFMiddleware(handler)
	}

	// Apply rate limiting if enabled
	var rateLimiter *middleware.RateLimiter
	if c.Config.RateLimit.Enabled {
		rateLimiter = middleware.NewRateLimiter(
			c.Config.RateLimit.Requests,
			c.Config.RateLimit.Window,
			c.Config.RateLimit.CleanupInterval,
		)
		c.RateLimiters = append(c.RateLimiters, rateLimiter)
		handler = middleware.RateLimitMiddleware(rateLimiter)(handler)
	}

	// Apply concurrency limiting if enabled
	if c.Config.Concurrency.Enabled {
		concurrencyLimiter := middleware.NewConcurrencyLimiter(c.Config.Concurrency.MaxConcurrent, c.Config.Concurrency.AcquireTimeout)
		handler = middleware.ConcurrencyLimitMiddleware(concurrencyLimiter)(handler)
	}

	return handler
}

// Shutdown stops all background goroutines and cleans up resources
func (c *Container) Shutdown() {
	// Stop all rate limiters to prevent goroutine leaks
	for _, rl := range c.RateLimiters {
		rl.Stop()
	}
	log.Println("[Container] All rate limiters stopped")

	// Close database connection if exists
	if c.DB != nil {
		if err := c.DB.Close(); err != nil {
			log.Printf("[Container] Error closing database: %v", err)
		} else {
			log.Println("[Container] Database connection closed")
		}
	}
}

// secureFileServer wraps http.FileServer to disable directory listing
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
