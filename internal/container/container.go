package container

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"s3-upload-tool/internal/auth"
	"s3-upload-tool/internal/config"
	"s3-upload-tool/internal/database"
	"s3-upload-tool/internal/handlers"
	"s3-upload-tool/internal/metrics"
	"s3-upload-tool/internal/middleware"
	"s3-upload-tool/internal/repository"
	"s3-upload-tool/internal/service"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Container struct {
	Config             *config.Config
	DB                 *database.DB
	S3Client           *s3.Client
	Uploader           *manager.Uploader
	S3Repository       repository.S3Repository
	UploadRepository   repository.UploadRepository
	UploadService      service.UploadService
	ChunkUploadService service.ChunkUploadService
	APIHandler         *handlers.APIHandler
	WPHandler          *handlers.WPHandler
	RepoFactory        *repository.RepositoryFactory
	ServiceFactory     *service.ServiceFactory
	RateLimiters       []*middleware.RateLimiter

	// Admin auth (Google OAuth + session) - chỉ khởi tạo đầy đủ khi DB bật,
	// vì session/admin_users đều cần bảng MySQL. AdminAuthHandler vẫn được
	// tạo dù thiếu Google Client ID/Secret - handler tự trả 503 rõ ràng.
	AdminUserRepository repository.AdminUserRepository
	SessionRepository   repository.SessionRepository
	SessionManager      *auth.SessionManager
	AdminAuthHandler    *handlers.AdminAuthHandler
	sessionCleanupStop  chan struct{}

	// Domain: artwork/award/dashboard/meta - chỉ khởi tạo đầy đủ khi DB bật
	// (cùng điều kiện với admin auth ở trên, vì mọi bảng domain đều ở MySQL).
	SchoolRepository        repository.SchoolRepository
	GradeLevelRepository    repository.GradeLevelRepository
	TopicCategoryRepository repository.TopicCategoryRepository
	StudentRepository       repository.StudentRepository
	ArtworkRepository       repository.ArtworkRepository
	AwardRepository         repository.AwardRepository
	ReactionRepository      repository.ReactionRepository
	CommentRepository       repository.CommentRepository
	ArtworkViewRepository   repository.ArtworkViewRepository
	DashboardRepository     repository.DashboardRepository

	ArtworkService       service.ArtworkService
	AwardService         service.AwardService
	TopicCategoryService service.TopicCategoryService
	DashboardService     service.DashboardService

	ArtworkHandler       *handlers.ArtworkHandler
	AwardHandler         *handlers.AwardHandler
	TopicCategoryHandler *handlers.TopicCategoryHandler
	DashboardHandler     *handlers.DashboardHandler
	MetaHandler          *handlers.MetaHandler
	PublicHandler        *handlers.PublicHandler
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

	// HTTP client for AWS SDK: no client-level Timeout (multipart uploads can exceed 60s).
	// Per-call context timeouts (UPLOAD_TIMEOUT_SECONDS) bound individual operations.
	httpClient := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:          500,
			MaxIdleConnsPerHost:   100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 60 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DisableKeepAlives:     false,
			DisableCompression:    false,
		},
	}

	// ! Timeout 30s để tránh hang khi khởi tạo AWS
	initCtx, initCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer initCancel()

	warnPlaceholderAWSCredentials()

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

	if resolved := resolveBucketRegion(initCtx, awsCfg, cfg.AWS.BucketName, cfg.AWS.Endpoint); resolved != "" && resolved != cfg.AWS.Region {
		log.Printf("[Container] S3 bucket region is %s (configured AWS_REGION=%s) — using bucket region", resolved, cfg.AWS.Region)
		cfg.AWS.Region = resolved
		awsCfg.Region = resolved
	}

	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.AWS.ForcePathStyle
		if cfg.AWS.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.AWS.Endpoint)
			o.UsePathStyle = true
		}
	})

	// * Multipart lên S3: part 8MB, concurrency 4 — tối ưu cho file ~20MB+
	uploader := manager.NewUploader(s3Client, func(u *manager.Uploader) {
		u.PartSize = 8 * 1024 * 1024
		u.Concurrency = 4
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

		if cfg.Database.AutoMigrate {
			migrateCtx, migrateCancel := context.WithTimeout(context.Background(), 2*time.Minute)
			migrateErr := database.RunMigrations(migrateCtx, db, "internal/database/migrations")
			migrateCancel()
			if migrateErr != nil {
				return nil, fmt.Errorf("failed to run database migrations: %w", migrateErr)
			}
		}

		uploadRepo, err = repository.NewUploadRepository(db)
		if err != nil {
			return nil, fmt.Errorf("failed to create upload repository: %w", err)
		}
	}

	// Admin auth (Google OAuth + session) - cần DB để lưu admin_users/admin_sessions.
	// AdminAuthHandler vẫn được tạo (dùng constructor trực tiếp, không qua
	// RepoFactory/ServiceFactory - factory đó chỉ dành cho S3/upload strategy)
	// ngay cả khi Google Client ID/Secret trống, để endpoint /auth/google/*
	// có thể trả 503 rõ ràng thay vì 404 khi chưa cấu hình.
	var adminUserRepo repository.AdminUserRepository
	var sessionRepo repository.SessionRepository
	var sessionMgr *auth.SessionManager
	var adminAuthHandler *handlers.AdminAuthHandler
	if db != nil {
		adminUserRepo = repository.NewAdminUserRepository(db)
		sessionRepo = repository.NewSessionRepository(db)
		sessionMgr = auth.NewSessionManager(sessionRepo, cfg.Auth)
		oauthConfig := auth.NewGoogleOAuthConfig(cfg.Auth)
		adminAuthHandler = handlers.NewAdminAuthHandler(
			oauthConfig,
			sessionMgr,
			adminUserRepo,
			cfg.Auth.AllowedEmailDomains,
			cfg.Auth.AllowedEmails,
			cfg.Auth.SecureCookie,
		)
	} else {
		log.Println("[Container] Database chưa bật - admin auth (Google OAuth) sẽ không khả dụng cho tới khi DATABASE_ENABLED=true")
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

	if cfg.AWS.BasePath != "" {
		log.Printf("[Container] S3 base path: %s/", cfg.AWS.BasePath)
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
		cfg.AWS.Endpoint,
		cfg.AWS.ForcePathStyle || cfg.AWS.Endpoint != "",
		cfg.AWS.BasePath,
	)
	if err != nil {
		return nil, err
	}

	chunkService := service.NewChunkUploadService(
		s3Repo,
		cfg.AWS.BucketName,
		cfg.AWS.Region,
		cfg.Directories.UploadDir,
		cfg.Upload.MaxSize,
		cfg.Upload.AbsoluteMaxSize,
		cfg.Upload.UploadTimeout,
		cfg.AWS.UseACL,
		cfg.AWS.UsePresignedURL,
		cfg.AWS.PresignedURLExpiry,
		cfg.AWS.Endpoint,
		cfg.AWS.ForcePathStyle || cfg.AWS.Endpoint != "",
		cfg.AWS.BasePath,
	)

	apiHandler := handlers.NewAPIHandler(
		uploadService,
		chunkService,
		s3Repo,
		cfg.Upload.MaxSize,
		cfg.Upload.AbsoluteMaxSize,
		db,
	)

	// Domain: artwork/award/dashboard/meta - cần DB (schools/grade_levels/
	// students/artworks/awards...). Dùng constructor trực tiếp (không qua
	// RepoFactory/ServiceFactory - factory đó chỉ dành cho S3/upload
	// strategy), theo đúng convention đã áp dụng cho admin auth ở trên.
	var (
		schoolRepo        repository.SchoolRepository
		gradeRepo         repository.GradeLevelRepository
		topicCategoryRepo repository.TopicCategoryRepository
		studentRepo       repository.StudentRepository
		artworkRepo       repository.ArtworkRepository
		awardRepo         repository.AwardRepository
		reactionRepo      repository.ReactionRepository
		commentRepo       repository.CommentRepository
		viewRepo          repository.ArtworkViewRepository
		dashboardRepo     repository.DashboardRepository
		artworkSvc        service.ArtworkService
		awardSvc          service.AwardService
		topicCategorySvc  service.TopicCategoryService
		dashboardSvc      service.DashboardService
		artworkHandler    *handlers.ArtworkHandler
		awardHandler      *handlers.AwardHandler
		topicCategoryHdlr *handlers.TopicCategoryHandler
		dashboardHdlr     *handlers.DashboardHandler
		metaHandler       *handlers.MetaHandler
		publicHandler     *handlers.PublicHandler
	)
	if db != nil {
		schoolRepo = repository.NewSchoolRepository(db)
		gradeRepo = repository.NewGradeLevelRepository(db)
		topicCategoryRepo = repository.NewTopicCategoryRepository(db)
		studentRepo = repository.NewStudentRepository(db)
		artworkRepo = repository.NewArtworkRepository(db)
		awardRepo = repository.NewAwardRepository(db)
		reactionRepo = repository.NewReactionRepository(db)
		commentRepo = repository.NewCommentRepository(db)
		viewRepo = repository.NewArtworkViewRepository(db)
		dashboardRepo = repository.NewDashboardRepository(db)

		artworkSvc = service.NewArtworkService(
			db, uploadService, artworkRepo, studentRepo, schoolRepo, gradeRepo, topicCategoryRepo, awardRepo, reactionRepo, commentRepo,
		)
		awardSvc = service.NewAwardService(awardRepo)
		topicCategorySvc = service.NewTopicCategoryService(topicCategoryRepo)
		dashboardSvc = service.NewDashboardService(dashboardRepo)

		artworkHandler = handlers.NewArtworkHandler(artworkSvc, cfg.Upload.MaxSize)
		awardHandler = handlers.NewAwardHandler(awardSvc)
		topicCategoryHdlr = handlers.NewTopicCategoryHandler(topicCategorySvc)
		dashboardHdlr = handlers.NewDashboardHandler(dashboardSvc)
		metaHandler = handlers.NewMetaHandler(schoolRepo, gradeRepo)
		publicHandler = handlers.NewPublicHandler(artworkSvc, reactionRepo, commentRepo, viewRepo, awardRepo)
	} else {
		log.Println("[Container] Database chưa bật - quản lý tác phẩm/giải thưởng/dashboard sẽ không khả dụng cho tới khi DATABASE_ENABLED=true")
	}

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

	c := &Container{
		Config:              cfg,
		DB:                  db,
		S3Client:            s3Client,
		Uploader:            uploader,
		S3Repository:        s3Repo,
		UploadRepository:    uploadRepo,
		UploadService:       uploadService,
		ChunkUploadService:  chunkService,
		APIHandler:          apiHandler,
		WPHandler:           wpHandler,
		RepoFactory:         repoFactory,
		ServiceFactory:      serviceFactory,
		AdminUserRepository: adminUserRepo,
		SessionRepository:   sessionRepo,
		SessionManager:      sessionMgr,
		AdminAuthHandler:    adminAuthHandler,

		SchoolRepository:        schoolRepo,
		GradeLevelRepository:    gradeRepo,
		TopicCategoryRepository: topicCategoryRepo,
		StudentRepository:       studentRepo,
		ArtworkRepository:       artworkRepo,
		AwardRepository:         awardRepo,
		ReactionRepository:      reactionRepo,
		CommentRepository:       commentRepo,
		ArtworkViewRepository:   viewRepo,
		DashboardRepository:     dashboardRepo,
		ArtworkService:          artworkSvc,
		AwardService:            awardSvc,
		TopicCategoryService:    topicCategorySvc,
		DashboardService:        dashboardSvc,
		ArtworkHandler:          artworkHandler,
		AwardHandler:            awardHandler,
		TopicCategoryHandler:    topicCategoryHdlr,
		DashboardHandler:        dashboardHdlr,
		MetaHandler:             metaHandler,
		PublicHandler:           publicHandler,
	}

	if sessionRepo != nil {
		c.startSessionCleanup()
	}

	return c, nil
}

// startSessionCleanup chạy goroutine nền dọn session hết hạn định kỳ mỗi
// giờ, theo đúng pattern RateLimiter.cleanup - dừng qua sessionCleanupStop
// trong Shutdown().
func (c *Container) startSessionCleanup() {
	c.sessionCleanupStop = make(chan struct{})
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				n, err := c.SessionRepository.DeleteExpired(ctx)
				cancel()
				if err != nil {
					log.Printf("[Container] Failed to clean up expired sessions: %v", err)
				} else if n > 0 {
					log.Printf("[Container] Cleaned up %d expired admin session(s)", n)
				}
			case <-c.sessionCleanupStop:
				return
			}
		}
	}()
}

func (c *Container) GetServerHandler() http.Handler {
	mux := http.NewServeMux()

	// API routes only (API is always enabled for API-only service)
	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/api/v1/upload", c.APIHandler.HandleUpload)
	apiMux.HandleFunc("/api/v1/upload-transaction", c.APIHandler.HandleUploadWithTransaction)
	apiMux.HandleFunc("/api/v1/upload/init", c.APIHandler.HandleChunkInit)
	apiMux.HandleFunc("/api/v1/upload/chunk", c.APIHandler.HandleChunkUpload)
	apiMux.HandleFunc("/api/v1/upload/complete", c.APIHandler.HandleChunkComplete)
	apiMux.HandleFunc("/api/v1/upload/abort", c.APIHandler.HandleChunkAbort)
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

	// Admin API routes: session-based auth (không dùng API key) - toàn bộ
	// /api/v1/admin/* đi qua AdminAuthMiddleware. Chỉ mount khi DB bật vì
	// AdminAuthHandler/SessionManager cần bảng admin_users/admin_sessions.
	if c.AdminAuthHandler != nil && c.SessionManager != nil {
		adminAPIMux := http.NewServeMux()
		adminAPIMux.HandleFunc("/api/v1/admin/auth/me", c.AdminAuthHandler.HandleMe)

		// Quản lý tác phẩm/giải thưởng/dashboard - chỉ có khi DB bật (cùng
		// điều kiện AdminAuthHandler != nil vì cả hai đều cần MySQL).
		if c.ArtworkHandler != nil {
			adminAPIMux.HandleFunc("POST /api/v1/admin/artworks/bulk-upload", c.ArtworkHandler.HandleBulkUpload)
			adminAPIMux.HandleFunc("POST /api/v1/admin/artworks", c.ArtworkHandler.HandleCreate)
			adminAPIMux.HandleFunc("GET /api/v1/admin/artworks", c.ArtworkHandler.HandleList)
			adminAPIMux.HandleFunc("GET /api/v1/admin/artworks/{id}", c.ArtworkHandler.HandleGet)
			adminAPIMux.HandleFunc("PUT /api/v1/admin/artworks/{id}", c.ArtworkHandler.HandleUpdate)
			adminAPIMux.HandleFunc("DELETE /api/v1/admin/artworks/{id}", c.ArtworkHandler.HandleDelete)
			adminAPIMux.HandleFunc("PATCH /api/v1/admin/artworks/{id}/featured", c.ArtworkHandler.HandleSetFeatured)
			adminAPIMux.HandleFunc("PATCH /api/v1/admin/artworks/bulk-featured", c.ArtworkHandler.HandleSetFeaturedBatch)
		}
		if c.AwardHandler != nil {
			adminAPIMux.HandleFunc("GET /api/v1/admin/awards", c.AwardHandler.HandleListAwards(true))
			adminAPIMux.HandleFunc("POST /api/v1/admin/awards", c.AwardHandler.HandleCreateAward)
			adminAPIMux.HandleFunc("PUT /api/v1/admin/awards/{id}", c.AwardHandler.HandleUpdateAward)
			adminAPIMux.HandleFunc("DELETE /api/v1/admin/awards/{id}", c.AwardHandler.HandleDeleteAward)
		}
		if c.TopicCategoryHandler != nil {
			adminAPIMux.HandleFunc("GET /api/v1/admin/topic-categories", c.TopicCategoryHandler.HandleListTopicCategories(true))
			adminAPIMux.HandleFunc("POST /api/v1/admin/topic-categories", c.TopicCategoryHandler.HandleCreateTopicCategory)
			adminAPIMux.HandleFunc("PUT /api/v1/admin/topic-categories/{id}", c.TopicCategoryHandler.HandleUpdateTopicCategory)
			adminAPIMux.HandleFunc("DELETE /api/v1/admin/topic-categories/{id}", c.TopicCategoryHandler.HandleDeleteTopicCategory)
		}
		if c.DashboardHandler != nil {
			adminAPIMux.HandleFunc("/api/v1/admin/dashboard/stats", c.DashboardHandler.HandleStats)
			adminAPIMux.HandleFunc("/api/v1/admin/dashboard/region-summary", c.DashboardHandler.HandleRegionSummary)
		}

		adminAPIHandler := middleware.AdminAuthMiddleware(c.SessionManager)(http.Handler(adminAPIMux))
		apiMux.Handle("/api/v1/admin/", adminAPIHandler)

		// Logout không bọc AdminAuthMiddleware: nếu session đã hết hạn phía
		// server thì vẫn phải cho phép request logout đi qua để xoá cookie
		// phía client sạch sẽ, tránh 401 vô nghĩa khi user chỉ muốn đăng xuất.
		apiMux.HandleFunc("/api/v1/admin/auth/logout", c.AdminAuthHandler.HandleLogout)
	}

	// Public read-only routes: không cần session/API key - dùng cho form
	// admin (chọn trường/khối) lẫn bộ lọc trang public.
	if c.MetaHandler != nil {
		apiMux.HandleFunc("/api/v1/schools", c.MetaHandler.HandleListSchools)
		apiMux.HandleFunc("/api/v1/grade-levels", c.MetaHandler.HandleListGradeLevels)
	}
	if c.AwardHandler != nil {
		apiMux.HandleFunc("/api/v1/awards", c.AwardHandler.HandleListAwards(false))
	}
	if c.TopicCategoryHandler != nil {
		apiMux.HandleFunc("/api/v1/topic-categories", c.TopicCategoryHandler.HandleListTopicCategories(false))
	}

	// Public API ẩn danh: reaction/comment/view cho trang "20 năm VASchools". Toàn
	// bộ route (GET lẫn POST/DELETE) đăng ký trên 1 mux duy nhất để tránh
	// đụng pattern khi 2 mux cùng khớp 1 path; rate limiter NGHIÊM HƠN
	// (20 req/phút/IP, tách biệt với rate limiter chung toàn API) chỉ áp
	// dụng cho method state-changing (POST/DELETE) qua điều kiện method
	// ngay trong middleware, GET không bị giới hạn riêng.
	if c.PublicHandler != nil {
		publicMux := http.NewServeMux()
		publicMux.HandleFunc("GET /api/v1/public/artworks", c.PublicHandler.HandleListArtworks)
		publicMux.HandleFunc("GET /api/v1/public/artworks/featured", c.PublicHandler.HandleListFeatured)
		publicMux.HandleFunc("GET /api/v1/public/artworks/{id}", c.PublicHandler.HandleGetArtwork)
		publicMux.HandleFunc("GET /api/v1/public/artworks/{id}/comments", c.PublicHandler.HandleListComments)
		publicMux.HandleFunc("GET /api/v1/public/billboard", c.PublicHandler.HandleBillboard)
		publicMux.HandleFunc("POST /api/v1/public/artworks/{id}/reactions", c.PublicHandler.HandleAddReaction)
		publicMux.HandleFunc("DELETE /api/v1/public/artworks/{id}/reactions/{type}", c.PublicHandler.HandleRemoveReaction)
		publicMux.HandleFunc("POST /api/v1/public/artworks/{id}/comments", c.PublicHandler.HandleCreateComment)
		publicMux.HandleFunc("DELETE /api/v1/public/artworks/{id}/comments/{commentID}", c.PublicHandler.HandleDeleteComment)

		var publicHandlerChain http.Handler = publicMux
		if c.Config.RateLimit.Enabled {
			publicRateLimiter := middleware.NewRateLimiter(20, time.Minute, c.Config.RateLimit.CleanupInterval)
			c.RateLimiters = append(c.RateLimiters, publicRateLimiter)
			strictLimit := middleware.RateLimitMiddleware(publicRateLimiter)(publicMux)
			publicHandlerChain = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost || r.Method == http.MethodDelete {
					strictLimit.ServeHTTP(w, r)
					return
				}
				publicMux.ServeHTTP(w, r)
			})
		}
		apiMux.Handle("/api/v1/public/", publicHandlerChain)
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

	// Google OAuth: browser-redirect flow, không phải JSON API - mount
	// ngoài /api/ để không bị CORS/API-key middleware của apiHandler áp vào.
	if c.AdminAuthHandler != nil {
		mux.HandleFunc("/auth/google/login", c.AdminAuthHandler.HandleGoogleLogin)
		mux.HandleFunc("/auth/google/callback", c.AdminAuthHandler.HandleGoogleCallback)
	}

	// Trang chia sẻ có Open Graph render phía server để Facebook lấy được
	// title/description/ảnh của đúng tác phẩm trước khi chuyển vào SPA.
	if c.PublicHandler != nil {
		mux.HandleFunc("GET /chia-se/tac-pham/{id}", c.PublicHandler.HandleArtworkSharePage)
	}

	// Serve WordPress uploads directory (with security)
	if c.Config.WordPress.Enabled {
		wpFs := secureFileServer(http.Dir(c.Config.Directories.WPUploadsDir))
		mux.Handle("/wp-content/uploads/", http.StripPrefix("/wp-content/uploads/", wpFs))
	}

	// SPA / static UI from web/dist when built; otherwise JSON API info
	mux.Handle("/", spaFileServer("web/dist"))

	handler := http.Handler(mux)

	// Gzip nằm trong cùng (sát mux nhất): nó cần thấy Content-Type do handler
	// đặt, và đặt trong cùng thì các middleware ngoài vẫn đo/ghi log bình thường.
	// Bundle SPA ~950KB JS + ~180KB CSS trước đây gửi thô hoàn toàn.
	handler = middleware.GzipMiddleware(handler)

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

	if c.ChunkUploadService != nil {
		c.ChunkUploadService.Stop()
		log.Println("[Container] Chunk upload sessions cleaned")
	}

	if c.sessionCleanupStop != nil {
		close(c.sessionCleanupStop)
		log.Println("[Container] Admin session cleanup stopped")
	}

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

func warnPlaceholderAWSCredentials() {
	key := strings.ToLower(strings.TrimSpace(os.Getenv("AWS_ACCESS_KEY_ID")))
	secret := strings.ToLower(strings.TrimSpace(os.Getenv("AWS_SECRET_ACCESS_KEY")))
	if key == "" || secret == "" {
		return
	}
	if strings.Contains(key, "your_access") || strings.Contains(secret, "your_secret") ||
		key == "changeme" || secret == "changeme" {
		log.Println("[Container] WARNING: placeholder AWS credentials detected in .env — S3 uploads will fail until real keys are set")
	}
}

// resolveBucketRegion asks S3 for the bucket location (via us-east-1) when no custom endpoint is set.
func resolveBucketRegion(ctx context.Context, awsCfg aws.Config, bucket, endpoint string) string {
	if bucket == "" || endpoint != "" {
		return ""
	}
	locator := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.Region = "us-east-1"
	})
	out, err := locator.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: aws.String(bucket),
	})
	if err != nil {
		log.Printf("[Container] Could not resolve bucket region for %s: %v", bucket, err)
		return ""
	}
	region := string(out.LocationConstraint)
	if region == "" {
		return "us-east-1"
	}
	return region
}

const apiInfoJSON = `{"service":"s3-upload-api","version":"1.0","endpoints":["/api/v1/upload","/api/v1/upload/init","/api/v1/upload/chunk","/api/v1/upload/complete","/api/v1/upload/abort","/api/v1/upload-transaction","/api/v1/health","/api/v1/metrics","/api/v1/wp-upload"]}`

// spaFileServer serves a Vite/React build with index.html fallback.
// If the dist directory is missing, returns API info JSON (dev-friendly).
func spaFileServer(distDir string) http.Handler {
	absDist, err := filepath.Abs(distDir)
	if err != nil {
		absDist = filepath.Clean(distDir)
	}
	indexPath := filepath.Join(absDist, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(apiInfoJSON))
		})
	}

	fileServer := http.FileServer(http.Dir(absDist))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			serveSPAIndex(w, r, indexPath)
			return
		}

		rel := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		if rel == "." || strings.HasPrefix(rel, "..") {
			serveSPAIndex(w, r, indexPath)
			return
		}
		full := filepath.Join(absDist, rel)
		absFull, err := filepath.Abs(full)
		if err != nil || (!strings.HasPrefix(absFull, absDist+string(os.PathSeparator)) && absFull != absDist) {
			serveSPAIndex(w, r, indexPath)
			return
		}
		if info, err := os.Stat(absFull); err == nil && !info.IsDir() {
			setStaticCacheHeader(w, r.URL.Path)
			fileServer.ServeHTTP(w, r)
			return
		}
		serveSPAIndex(w, r, indexPath)
	})
}

// serveSPAIndex trả index.html cho mọi route của SPA.
//
// index.html KHÔNG được cache lâu: nó chứa tên file bundle đã băm, nên bản cũ
// nằm trong cache sẽ trỏ mãi vào bundle của lần deploy trước. no-cache vẫn cho
// phép dùng lại sau khi revalidate (304), chỉ bắt buộc hỏi lại server mỗi lần.
func serveSPAIndex(w http.ResponseWriter, r *http.Request, indexPath string) {
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, indexPath)
}

// setStaticCacheHeader đặt thời gian cache theo loại tài nguyên.
//
// Trước đây không có Cache-Control nào, nên trình duyệt phải revalidate cả
// bundle đã băm tên ở mỗi lần tải trang.
func setStaticCacheHeader(w http.ResponseWriter, urlPath string) {
	switch {
	// Vite băm nội dung vào tên file (index-B5t9s4PZ.js), nên nội dung tại một
	// URL không bao giờ đổi - đổi nội dung là đổi luôn tên file. immutable báo
	// trình duyệt đừng revalidate kể cả khi người dùng bấm tải lại.
	case strings.HasPrefix(urlPath, "/assets/"):
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	// Ảnh/font tĩnh không băm tên: cache 7 ngày rồi revalidate, đủ để đổi ảnh
	// mà không cần đợi quá lâu.
	case strings.HasPrefix(urlPath, "/images/"), strings.HasPrefix(urlPath, "/fonts/"):
		w.Header().Set("Cache-Control", "public, max-age=604800")
	default:
		w.Header().Set("Cache-Control", "public, max-age=3600")
	}
}
