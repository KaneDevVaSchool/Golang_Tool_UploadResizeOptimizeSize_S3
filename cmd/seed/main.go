// Package main triển khai cmd/seed - công cụ dọn sạch dữ liệu nghiệp vụ và
// nạp lại dữ liệu demo (trường/khối lớp/học sinh/tác phẩm/giải/nhóm chủ đề/
// tương tác mẫu) để dựng môi trường demo hoặc kiểm thử thủ công nhanh.
//
// Không đụng admin_users/admin_sessions (tài khoản đăng nhập không phải dữ
// liệu demo) và không đụng grade_levels (12 khối lớp là danh mục cố định,
// không có gì để làm mới). schools ĐƯỢC ghi đè bằng danh sách 16 cơ sở thật
// của hệ thống Việt Mỹ - xem lý do ở seedSchools().
//
// Chạy: go run ./cmd/seed --images="C:\Users\ASUS\Desktop\theme"
// An toàn: yêu cầu gõ đúng "XOA" để xác nhận trước khi đụng dữ liệu, trừ khi
// truyền --yes (dùng cho chạy không tương tác, vd CI).
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"s3-upload-tool/internal/config"
	"s3-upload-tool/internal/database"
	"s3-upload-tool/internal/repository"
	"s3-upload-tool/internal/service"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
	var (
		imagesDir = flag.String("images", `C:\Users\ASUS\Desktop\theme`, "Thư mục chứa ảnh dùng để seed tác phẩm (mỗi ảnh = 1 tác phẩm, không lặp lại)")
		yes       = flag.Bool("yes", false, "Bỏ qua xác nhận tương tác (dùng khi chạy không có terminal, vd script)")
	)
	flag.Parse()

	_ = config.LoadEnvFile()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Không tải được cấu hình: %v", err)
	}
	if !cfg.Database.Enabled || cfg.Database.DataSource == "" {
		log.Fatalf("DATABASE_ENABLED phải là true và DATABASE_URL phải có giá trị để chạy seed")
	}

	entries, err := loadImageFiles(*imagesDir)
	if err != nil {
		log.Fatalf("Không đọc được thư mục ảnh %s: %v", *imagesDir, err)
	}
	if len(entries) == 0 {
		log.Fatalf("Thư mục %s không có ảnh nào (.jpg/.jpeg/.png)", *imagesDir)
	}
	log.Printf("Tìm thấy %d ảnh trong %s", len(entries), *imagesDir)

	if !*yes {
		confirmOrExit()
	}

	db, err := database.NewDB(database.Config{
		Driver:      cfg.Database.Driver,
		DataSource:  cfg.Database.DataSource,
		MaxOpen:     cfg.Database.MaxOpen,
		MaxIdle:     cfg.Database.MaxIdle,
		MaxLifetime: time.Duration(cfg.Database.MaxLifetime) * time.Second,
	})
	if err != nil {
		log.Fatalf("Không kết nối được MySQL: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	s3Client, uploader, err := newS3Client(ctx, cfg)
	if err != nil {
		log.Fatalf("Không khởi tạo được S3 client: %v", err)
	}

	// Dựng lại đúng dây chuyền repository/service như container.go, nhưng
	// chỉ phần seed script này cần - không kéo theo handler/middleware.
	s3Repo := repository.NewS3Repository(uploader, s3Client, cfg.AWS.BucketName)
	uploadSvc := service.NewUploadService(
		s3Repo,
		cfg.AWS.BucketName, cfg.AWS.Region, cfg.Directories.UploadDir,
		cfg.Upload.MaxSize, cfg.Upload.UploadTimeout,
		cfg.AWS.UseACL, cfg.AWS.UsePresignedURL, cfg.AWS.PresignedURLExpiry,
		cfg.AWS.Endpoint, cfg.AWS.ForcePathStyle || cfg.AWS.Endpoint != "", cfg.AWS.BasePath,
	)

	schoolRepo := repository.NewSchoolRepository(db)
	gradeRepo := repository.NewGradeLevelRepository(db)
	topicRepo := repository.NewTopicCategoryRepository(db)
	studentRepo := repository.NewStudentRepository(db)
	artworkRepo := repository.NewArtworkRepository(db)
	awardRepo := repository.NewAwardRepository(db)
	reactionRepo := repository.NewReactionRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	downloadRepo := repository.NewArtworkDownloadRepository(db)

	artworkSvc := service.NewArtworkService(
		db, uploadSvc, artworkRepo, studentRepo, schoolRepo, gradeRepo, topicRepo, awardRepo, reactionRepo, commentRepo, downloadRepo,
	)

	s := &seeder{
		ctx:          ctx,
		cfg:          cfg,
		db:           db,
		s3Repo:       s3Repo,
		schoolRepo:   schoolRepo,
		gradeRepo:    gradeRepo,
		topicRepo:    topicRepo,
		awardRepo:    awardRepo,
		reactionRepo: reactionRepo,
		commentRepo:  commentRepo,
		artworkSvc:   artworkSvc,
	}

	log.Println("=== Bước 1/4: Xoá sạch dữ liệu nghiệp vụ cũ ===")
	if err := s.wipe(); err != nil {
		log.Fatalf("Xoá dữ liệu cũ thất bại: %v", err)
	}

	log.Println("=== Bước 2/4: Nạp lại danh mục (trường/giải/nhóm chủ đề) ===")
	schools, err := s.seedSchools()
	if err != nil {
		log.Fatalf("Seed schools thất bại: %v", err)
	}
	grades, err := gradeRepo.List(ctx)
	if err != nil {
		log.Fatalf("Không đọc được grade_levels: %v", err)
	}
	awards, err := s.seedAwards(grades)
	if err != nil {
		log.Fatalf("Seed awards thất bại: %v", err)
	}
	topics, err := s.seedTopicCategories()
	if err != nil {
		log.Fatalf("Seed topic_categories thất bại: %v", err)
	}

	log.Println("=== Bước 3/4: Upload ảnh + tạo tác phẩm ===")
	artworks, err := s.seedArtworks(entries, schools, grades, topics, awards)
	if err != nil {
		log.Fatalf("Seed artworks thất bại: %v", err)
	}

	log.Println("=== Bước 4/4: Nạp reaction/comment mẫu ===")
	if err := s.seedEngagement(artworks); err != nil {
		log.Fatalf("Seed reaction/comment thất bại: %v", err)
	}

	log.Printf("Hoàn tất: %d trường, %d giải, %d nhóm chủ đề, %d tác phẩm.", len(schools), len(awards), len(topics), len(artworks))
}

func confirmOrExit() {
	fmt.Println("!!! CẢNH BÁO: thao tác này XOÁ SẠCH artworks/students/awards/topic_categories/")
	fmt.Println("reaction/comment/view hiện có trong DB, xoá object S3 tương ứng, và ghi đè")
	fmt.Println("danh sách schools bằng 16 cơ sở thật. admin_users/admin_sessions/grade_levels")
	fmt.Println("KHÔNG bị đụng tới. Không thể hoàn tác.")
	fmt.Print(`Gõ "XOA" để xác nhận: `)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	if strings.TrimSpace(line) != "XOA" {
		fmt.Println("Huỷ - không có gì bị thay đổi.")
		os.Exit(1)
	}
}

func newS3Client(ctx context.Context, cfg *config.Config) (*s3.Client, *manager.Uploader, error) {
	initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	awsCfg, err := awsconfig.LoadDefaultConfig(initCtx, awsconfig.WithRegion(cfg.AWS.Region))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.AWS.ForcePathStyle
		if cfg.AWS.Endpoint != "" {
			o.BaseEndpoint = awssdk.String(cfg.AWS.Endpoint)
			o.UsePathStyle = true
		}
	})
	uploader := manager.NewUploader(client, func(u *manager.Uploader) {
		u.PartSize = 8 * 1024 * 1024
		u.Concurrency = 4
	})
	return client, uploader, nil
}

// imageFile là 1 ảnh nguồn từ thư mục theme, đọc sẵn vào RAM một lần vì mỗi
// ảnh chỉ dùng đúng 1 lần (không lặp lại) và tập ảnh đủ nhỏ (77 file, vài MB).
type imageFile struct {
	Name string
	Path string
	Data []byte
}

var allowedImageExt = map[string]bool{".jpg": true, ".jpeg": true, ".png": true}

func loadImageFiles(dir string) ([]imageFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []imageFile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if !allowedImageExt[ext] {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("không đọc được %s: %w", path, err)
		}
		files = append(files, imageFile{Name: e.Name(), Path: path, Data: data})
	}

	// Sắp theo tên để lần seed nào cũng gán ảnh -> tác phẩm giống nhau,
	// dễ đối chiếu khi debug.
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	return files, nil
}

// contentTypeFor suy content-type từ đuôi file - dùng http.DetectContentType
// làm phương án chính vì đây là cách duy nhất đọc magic byte thật (đúng
// nguyên tắc "không chỉ tin đuôi file" của dự án), phần mở rộng chỉ là dự
// phòng khi DetectContentType không nhận ra (ảnh hỏng, dữ liệu lạ).
func contentTypeFor(data []byte, name string) string {
	if ct := http.DetectContentType(data); strings.HasPrefix(ct, "image/") {
		return ct
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png":
		return "image/png"
	default:
		return "image/jpeg"
	}
}

// randomFrom trả 1 phần tử ngẫu nhiên trong slice không rỗng.
func randomFrom[T any](items []T) T {
	return items[rand.Intn(len(items))]
}
