package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"s3-upload-tool/internal/database"
	"s3-upload-tool/internal/models"
	"s3-upload-tool/internal/repository"
	"s3-upload-tool/internal/utils"
)

// BulkUploadItem là kết quả 1 file trong bulk-upload - chỉ chứa thông tin
// S3 (chưa ghi DB artworks). FE dùng tempKey để khớp lại file gốc với form
// nhập metadata; s3Key/s3URL dùng khi gọi CreateArtworkFromUpload.
type BulkUploadItem struct {
	TempKey  string `json:"temp_key"`
	FileName string `json:"file_name"`
	S3Key    string `json:"s3_key,omitempty"`
	S3URL    string `json:"s3_url,omitempty"`
	FileSize int64  `json:"file_size,omitempty"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	// Variants là các cỡ ảnh nhỏ đã sinh kèm lúc upload (thumb/medium/large
	// × webp/jpg). FE gửi trả lại nguyên vẹn ở bước tạo artwork để lưu vào DB.
	Variants models.ArtworkVariants `json:"variants,omitempty"`
	Error    string                 `json:"error,omitempty"`
}

// BulkUploadFile là 1 file đầu vào cho BulkUploadToS3 - TempKey do FE tự
// sinh (vd id preview item) để khớp lại kết quả, không phải ID thật trong DB.
type BulkUploadFile struct {
	TempKey  string
	FileName string
	Reader   io.ReadSeeker
	FileSize int64
}

// CreateArtworkRequest là input để tạo 1 artwork record (bước 2, sau khi
// ảnh đã có sẵn trên S3 từ BulkUploadToS3 hoặc upload đơn).
type CreateArtworkRequest struct {
	Title           string
	StudentName     string
	SchoolID        int64
	GradeLevelID    int64
	TopicCategoryID *int64
	ClassName       string
	S3Key           string
	S3URL           string
	FileSize        int64
	Width           int
	Height          int
	Variants        models.ArtworkVariants
	// AwardIDs là danh sách giải gán ngay lúc tạo - một tác phẩm có thể nhận
	// nhiều giải cùng lúc (vd giải chính Nhất/Nhì/Ba + giải Đặc biệt phụ).
	AwardIDs  []int64
	CreatedBy *int64
}

// UpdateArtworkRequest là input để cập nhật metadata artwork - KHÔNG đổi
// lại file S3 (đổi ảnh nghĩa là xoá tác phẩm cũ, tạo tác phẩm mới).
type UpdateArtworkRequest struct {
	Title           string
	StudentID       int64
	SchoolID        int64
	GradeLevelID    int64
	TopicCategoryID *int64
	IsFeatured      bool
	IsPublished     bool
	// AwardIDs: nil = không đổi giải hiện tại; []int64{} (rỗng, không nil) =
	// gỡ hết giải; danh sách khác rỗng = thay toàn bộ giải hiện tại bằng
	// danh sách này. Không còn giới hạn 1 giải/tác phẩm - schema artwork_awards
	// vốn đã N:N, giới hạn cũ chỉ do UI single-select áp lên.
	AwardIDs []int64
}

// ArtworkListResult gộp danh sách + tổng số + reaction/comment counts thành
// 1 DTO trả về cho handler, tránh N+1 query khi FE cần hiện đủ metadata
// (giải thưởng, lượt reaction, lượt comment) ngay trên trang danh sách.
type ArtworkListResult struct {
	Items      []*models.ArtworkWithMeta
	TotalCount int64
	Page       int
	PageSize   int
}

// SitemapArtwork là dạng rút gọn của một tác phẩm, chỉ đủ dữ liệu để build
// sitemap.xml (URL + lastmod) - không kéo theo tên học sinh/trường/giải vì
// sitemap không cần tới.
type SitemapArtwork struct {
	ID        int64
	UpdatedAt time.Time
}

// ArtworkService là business logic cho quản lý tác phẩm - tái dùng
// UploadService.UploadImage nguyên bản cho phần đẩy file lên S3, không viết
// lại logic S3 (validate/temp-file/multipart).
type ArtworkService interface {
	// BulkUploadToS3 upload nhiều file song song (giới hạn concurrency),
	// CHỈ đẩy lên S3 - không ghi bảng artworks. Lỗi từng file không làm fail
	// cả batch, item lỗi có Error != "" trong kết quả.
	BulkUploadToS3(ctx context.Context, files []BulkUploadFile, maxSize int64) []BulkUploadItem

	CreateArtworkFromUpload(ctx context.Context, req CreateArtworkRequest) (*models.Artwork, error)
	UpdateArtwork(ctx context.Context, id int64, req UpdateArtworkRequest) (*models.Artwork, error)
	// DeleteArtwork xoá bản ghi DB (CASCADE reaction/comment/view/giải) và
	// object S3 (ảnh gốc + biến thể/thumbnail lưu trong variants).
	DeleteArtwork(ctx context.Context, id int64) error
	// DeleteArtworkBatch xoá nhiều tác phẩm cùng lúc (thao tác bulk ở trang
	// quản trị). Lỗi xoá S3 của một tác phẩm không chặn việc xoá các tác phẩm
	// còn lại - chỉ ghi log, vì bản ghi DB mới là thứ người dùng cần thấy biến
	// mất ngay; ảnh mồ côi trên S3 xử lý bằng công cụ dọn định kỳ riêng (xem
	// docs/plan/02-roadmap.md P2.2). Trả về số tác phẩm đã xoá được bản ghi DB.
	DeleteArtworkBatch(ctx context.Context, ids []int64) (int, error)
	GetArtwork(ctx context.Context, id int64) (*models.ArtworkWithMeta, error)
	ListArtworks(ctx context.Context, filter models.ArtworkFilter) (*ArtworkListResult, error)
	// ListPublishedForSitemap trả về MỌI tác phẩm đã publish, chỉ gồm ID và
	// thời điểm cập nhật - đủ cho sitemap.xml (loc + lastmod), KHÔNG enrich
	// (không cần tên học sinh/trường/giải) và tự lặp trang để vượt trần
	// page_size=100 của ArtworkRepository.List - trần đó bảo vệ endpoint
	// public/admin khỏi bị lạm dụng page_size lớn, sitemap tự lo lặp ở đây
	// thay vì nới trần dùng chung. Xem PublicHandler.HandleSitemap.
	ListPublishedForSitemap(ctx context.Context) ([]SitemapArtwork, error)
	SetFeatured(ctx context.Context, id int64, featured bool) error
	// SetFeaturedBatch bật/tắt tiêu biểu hàng loạt cho thao tác bulk ở trang
	// quản trị - 1 câu UPDATE thay vì N request PATCH đơn lẻ từ frontend.
	SetFeaturedBatch(ctx context.Context, ids []int64, featured bool) error
	// LogDownload ghi nhật ký một lượt tải ảnh gốc (admin hoặc public) để
	// truy vết khi ảnh bị phát tán sai mục đích. Đây là thao tác phụ trợ:
	// lỗi ghi log KHÔNG được chặn việc tải, nên chỉ trả lỗi để handler tự
	// quyết định log cảnh báo - xem ArtworkHandler.HandleDownload và
	// PublicHandler.HandleDownloadArtwork.
	LogDownload(ctx context.Context, download *models.ArtworkDownload) error
}

type artworkService struct {
	db                *database.DB
	uploadService     UploadService
	artworkRepo       repository.ArtworkRepository
	studentRepo       repository.StudentRepository
	schoolRepo        repository.SchoolRepository
	gradeRepo         repository.GradeLevelRepository
	topicCategoryRepo repository.TopicCategoryRepository
	awardRepo         repository.AwardRepository
	reactionRepo      repository.ReactionRepository
	commentRepo       repository.CommentRepository
	downloadRepo      repository.ArtworkDownloadRepository
	bulkConcurrency   int

	// Cache trường + khối lớp + nhóm chủ đề. Đây là các bảng tham chiếu tĩnh
	// (5, 12, và vài chục dòng, seed sẵn hoặc admin hiếm khi đổi) nhưng
	// enrichArtworks lại nạp TOÀN BỘ ở mọi lần gọi - kể cả khi chỉ lấy chi
	// tiết một tác phẩm. Giữ trong tiến trình với TTL ngắn: đủ để thay đổi
	// hiếm hoi tự hiện ra sau vài phút mà không cần khởi động lại.
	refMu                 sync.RWMutex
	refExpiresAt          time.Time
	cachedSchools         map[int64]*models.School
	cachedGrades          map[int64]*models.GradeLevel
	cachedTopicCategories map[int64]*models.TopicCategory
}

// refCacheTTL - đủ ngắn để thêm trường/khối mới tự xuất hiện, đủ dài để loại
// hẳn hai truy vấn này khỏi đường đi của mọi request đọc.
const refCacheTTL = 5 * time.Minute

func NewArtworkService(
	db *database.DB,
	uploadService UploadService,
	artworkRepo repository.ArtworkRepository,
	studentRepo repository.StudentRepository,
	schoolRepo repository.SchoolRepository,
	gradeRepo repository.GradeLevelRepository,
	topicCategoryRepo repository.TopicCategoryRepository,
	awardRepo repository.AwardRepository,
	reactionRepo repository.ReactionRepository,
	commentRepo repository.CommentRepository,
	downloadRepo repository.ArtworkDownloadRepository,
) ArtworkService {
	return &artworkService{
		db:                db,
		uploadService:     uploadService,
		artworkRepo:       artworkRepo,
		studentRepo:       studentRepo,
		schoolRepo:        schoolRepo,
		gradeRepo:         gradeRepo,
		topicCategoryRepo: topicCategoryRepo,
		awardRepo:         awardRepo,
		reactionRepo:      reactionRepo,
		commentRepo:       commentRepo,
		downloadRepo:      downloadRepo,
		bulkConcurrency:   5, // giới hạn upload song song, tránh áp đảo S3/mạng khi admin chọn hàng chục ảnh cùng lúc
	}
}

// LogDownload xem interface ArtworkService - thao tác phụ trợ, chỉ INSERT.
func (s *artworkService) LogDownload(ctx context.Context, download *models.ArtworkDownload) error {
	if s.downloadRepo == nil {
		return fmt.Errorf("chưa cấu hình repository nhật ký tải ảnh")
	}
	return s.downloadRepo.Create(ctx, download)
}

func (s *artworkService) BulkUploadToS3(ctx context.Context, files []BulkUploadFile, maxSize int64) []BulkUploadItem {
	results := make([]BulkUploadItem, len(files))
	sem := make(chan struct{}, s.bulkConcurrency)
	var wg sync.WaitGroup

	for i, f := range files {
		wg.Add(1)
		go func(idx int, file BulkUploadFile) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			item := BulkUploadItem{TempKey: file.TempKey, FileName: file.FileName}

			width, height, ok := utils.DecodeImageDimensions(file.Reader)
			if ok {
				item.Width = width
				item.Height = height
			}
			if _, err := file.Reader.Seek(0, io.SeekStart); err != nil {
				item.Error = "không thể đọc lại file sau khi phân tích kích thước ảnh"
				results[idx] = item
				return
			}

			resp, err := s.uploadService.UploadImage(ctx, file.FileName, file.Reader, file.FileSize, maxSize)
			if err != nil {
				log.Printf("[ArtworkService] Bulk upload lỗi cho %s: %v", file.FileName, err)
				item.Error = err.Error()
				results[idx] = item
				return
			}

			item.S3Key = resp.Key
			item.S3URL = resp.URL
			item.FileSize = file.FileSize
			item.Variants = s.buildVariants(ctx, file, resp.Key)
			results[idx] = item
		}(i, f)
	}

	wg.Wait()
	return results
}

// variantContentTypes - S3 phục vụ object kèm đúng Content-Type này, nếu sai
// thì trình duyệt từ chối hiển thị hoặc tải về thay vì render.
var variantContentTypes = map[string]string{
	"webp": "image/webp",
	"jpg":  "image/jpeg",
}

// buildVariants sinh các cỡ ảnh nhỏ rồi đẩy lên S3 cạnh ảnh gốc.
//
// Cố ý KHÔNG làm hỏng cả lần upload khi sinh biến thể thất bại: ảnh gốc đã
// nằm an toàn trên S3, và trang vẫn hiển thị được (chỉ là tải nặng hơn) nhờ
// đường lui về image_url ở frontend. Bắt người dùng upload lại từ đầu chỉ vì
// khâu tối ưu phụ trợ hỏng thì thiệt hơn nhiều.
func (s *artworkService) buildVariants(ctx context.Context, file BulkUploadFile, originalKey string) models.ArtworkVariants {
	if _, err := file.Reader.Seek(0, io.SeekStart); err != nil {
		log.Printf("[ArtworkService] Bỏ qua sinh biến thể cho %s: không tua lại được file: %v", file.FileName, err)
		return nil
	}

	original, err := io.ReadAll(file.Reader)
	if err != nil {
		log.Printf("[ArtworkService] Bỏ qua sinh biến thể cho %s: không đọc được file: %v", file.FileName, err)
		return nil
	}

	generated, _, _, err := GenerateVariants(ctx, original)
	if err != nil {
		log.Printf("[ArtworkService] Không sinh được biến thể cho %s: %v", file.FileName, err)
		return nil
	}
	if len(generated) == 0 {
		// Ảnh gốc vốn đã nhỏ hơn mọi cỡ đích - dùng thẳng ảnh gốc là đúng nhất.
		return nil
	}

	// Bỏ đuôi file của key gốc rồi nối hậu tố biến thể:
	//   vaschools-uploads/abc123.jpg -> vaschools-uploads/abc123_thumb.webp
	base := strings.TrimSuffix(originalKey, filepath.Ext(originalKey))

	var (
		mu       sync.Mutex
		variants = make(models.ArtworkVariants, len(generated))
		wg       sync.WaitGroup
	)

	for _, v := range generated {
		wg.Add(1)
		go func(v GeneratedVariant) {
			defer wg.Done()

			key := fmt.Sprintf("%s_%s", base, v.Key())
			url, err := s.uploadService.UploadDerived(ctx, key, v.Data, variantContentTypes[v.Format])
			if err != nil {
				log.Printf("[ArtworkService] Upload biến thể %s lỗi: %v", key, err)
				return
			}

			mu.Lock()
			variants[fmt.Sprintf("%s_%s", v.Name, v.Format)] = url
			mu.Unlock()
		}(v)
	}

	wg.Wait()

	if len(variants) == 0 {
		return nil
	}
	return variants
}

func (s *artworkService) CreateArtworkFromUpload(ctx context.Context, req CreateArtworkRequest) (*models.Artwork, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	var txErr error
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
		if txErr != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("[ArtworkService] Failed to rollback: %v (original error: %v)", rbErr, txErr)
			}
		}
	}()

	var className *string
	if req.ClassName != "" {
		className = &req.ClassName
	}

	student, err := s.studentRepo.Create(ctx, tx, &models.Student{
		FullName:     req.StudentName,
		SchoolID:     req.SchoolID,
		GradeLevelID: req.GradeLevelID,
		ClassName:    className,
	})
	if err != nil {
		txErr = err
		return nil, fmt.Errorf("failed to create student: %w", err)
	}

	artwork := &models.Artwork{
		Title:           req.Title,
		StudentID:       student.ID,
		SchoolID:        req.SchoolID,
		GradeLevelID:    req.GradeLevelID,
		TopicCategoryID: req.TopicCategoryID,
		S3Key:           req.S3Key,
		S3URL:           req.S3URL,
		Variants:        req.Variants,
		FileSize:        req.FileSize,
		IsPublished:     true,
		CreatedBy:       req.CreatedBy,
	}
	// thumbnail_url vẫn được ghi song song với variants: trang admin và các
	// component cũ đọc trường này, và nó là bước lui một nấc trước khi phải
	// rơi về ảnh gốc. Ưu tiên bản JPEG vì đây là đường dự phòng - mọi trình
	// duyệt đều đọc được.
	if thumb := req.Variants["thumb_jpg"]; thumb != "" {
		artwork.ThumbnailURL = &thumb
	} else if thumb := req.Variants["thumb_webp"]; thumb != "" {
		artwork.ThumbnailURL = &thumb
	}
	if req.Width > 0 {
		artwork.Width = &req.Width
	}
	if req.Height > 0 {
		artwork.Height = &req.Height
	}

	created, err := s.artworkRepo.Create(ctx, tx, artwork)
	if err != nil {
		txErr = err
		return nil, fmt.Errorf("failed to create artwork: %w", err)
	}

	if err := tx.Commit(); err != nil {
		txErr = err
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Gán giải (nếu có) sau khi commit - artwork_awards có FK riêng, không
	// bắt buộc phải cùng transaction với việc tạo artwork. Một tác phẩm có
	// thể nhận nhiều giải cùng lúc (giải chính + giải Đặc biệt phụ).
	for _, awardID := range req.AwardIDs {
		if err := s.awardRepo.AttachToArtwork(ctx, created.ID, awardID); err != nil {
			log.Printf("[ArtworkService] Tạo artwork thành công nhưng gán giải thất bại (id=%d, award=%d): %v", created.ID, awardID, err)
		}
	}

	return created, nil
}

func (s *artworkService) UpdateArtwork(ctx context.Context, id int64, req UpdateArtworkRequest) (*models.Artwork, error) {
	existing, err := s.artworkRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get artwork: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("artwork not found: id=%d", id)
	}

	existing.Title = req.Title
	existing.StudentID = req.StudentID
	existing.SchoolID = req.SchoolID
	existing.GradeLevelID = req.GradeLevelID
	existing.TopicCategoryID = req.TopicCategoryID
	existing.IsFeatured = req.IsFeatured
	existing.IsPublished = req.IsPublished

	if err := s.artworkRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to update artwork: %w", err)
	}

	// req.AwardIDs nil = giữ nguyên giải hiện tại (client không gửi trường
	// này). Khác rỗng hay không đều là "đặt lại toàn bộ danh sách giải":
	// gỡ hết giải cũ rồi gắn đúng danh sách mới - đơn giản hơn diff từng
	// phần tử, và số giải mỗi tác phẩm luôn nhỏ nên không đáng lo hiệu năng.
	if req.AwardIDs != nil {
		current, err := s.awardRepo.ListByArtworkID(ctx, id)
		if err != nil {
			log.Printf("[ArtworkService] Không đọc được giải hiện tại của artwork %d: %v", id, err)
		} else {
			for _, a := range current {
				if err := s.awardRepo.DetachFromArtwork(ctx, id, a.ID); err != nil {
					log.Printf("[ArtworkService] Gỡ giải cũ thất bại (artwork=%d, award=%d): %v", id, a.ID, err)
				}
			}
		}
		for _, awardID := range req.AwardIDs {
			if awardID <= 0 {
				continue
			}
			if err := s.awardRepo.AttachToArtwork(ctx, id, awardID); err != nil {
				log.Printf("[ArtworkService] Gán giải mới thất bại (artwork=%d, award=%d): %v", id, awardID, err)
			}
		}
	}

	return existing, nil
}

func (s *artworkService) DeleteArtwork(ctx context.Context, id int64) error {
	artwork, err := s.artworkRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get artwork: %w", err)
	}
	if artwork == nil {
		return fmt.Errorf("artwork not found: id=%d", id)
	}

	for _, key := range collectArtworkS3Keys(artwork, s.uploadService.ObjectKeyFromURL) {
		if err := s.uploadService.DeleteObject(ctx, key); err != nil {
			return fmt.Errorf("failed to delete S3 object %s: %w", key, err)
		}
	}

	return s.artworkRepo.Delete(ctx, id)
}

// DeleteArtworkBatch xoá từng tác phẩm một (không phải 1 câu SQL IN(...) như
// SetFeaturedBatch) vì mỗi tác phẩm còn cần xoá kèm object S3 riêng - không
// gộp được thành 1 lệnh. Tác phẩm nào xoá S3 lỗi thì GIỮ NGUYÊN bản ghi DB
// (giống DeleteArtwork đơn lẻ) và bỏ qua sang tác phẩm tiếp theo, thay vì làm
// hỏng cả lô: admin chọn 50 ảnh xoá, 1 ảnh lỗi mạng lúc gọi S3 không nên khiến
// 49 ảnh còn lại cũng không xoá được.
func (s *artworkService) DeleteArtworkBatch(ctx context.Context, ids []int64) (int, error) {
	if len(ids) == 0 {
		return 0, fmt.Errorf("chưa chọn tác phẩm nào")
	}

	deleted := 0
	for _, id := range ids {
		if err := s.DeleteArtwork(ctx, id); err != nil {
			log.Printf("[ArtworkService] Xoá tác phẩm %d trong lô thất bại: %v", id, err)
			continue
		}
		deleted++
	}
	return deleted, nil
}

// collectArtworkS3Keys gom key ảnh gốc và mọi biến thể (từ variants/thumbnail).
func collectArtworkS3Keys(a *models.Artwork, keyFromURL func(string) string) []string {
	seen := make(map[string]struct{})
	var keys []string
	add := func(k string) {
		k = strings.TrimSpace(k)
		if k == "" {
			return
		}
		if _, ok := seen[k]; ok {
			return
		}
		seen[k] = struct{}{}
		keys = append(keys, k)
	}

	add(a.S3Key)
	if a.Variants != nil {
		for _, objectURL := range a.Variants {
			add(keyFromURL(objectURL))
		}
	}
	if a.ThumbnailURL != nil {
		add(keyFromURL(*a.ThumbnailURL))
	}
	return keys
}

func (s *artworkService) GetArtwork(ctx context.Context, id int64) (*models.ArtworkWithMeta, error) {
	artwork, err := s.artworkRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get artwork: %w", err)
	}
	if artwork == nil {
		return nil, nil
	}

	result, err := s.enrichArtworks(ctx, []*models.Artwork{artwork})
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}
	return result[0], nil
}

func (s *artworkService) ListArtworks(ctx context.Context, filter models.ArtworkFilter) (*ArtworkListResult, error) {
	artworks, total, err := s.artworkRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list artworks: %w", err)
	}

	enriched, err := s.enrichArtworks(ctx, artworks)
	if err != nil {
		return nil, err
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	return &ArtworkListResult{
		Items:      enriched,
		TotalCount: total,
		Page:       page,
		PageSize:   pageSize,
	}, nil
}

func (s *artworkService) ListPublishedForSitemap(ctx context.Context) ([]SitemapArtwork, error) {
	published := true
	const pageSize = 100 // khớp trần đã có ở ArtworkRepository.List

	var result []SitemapArtwork
	for page := 1; ; page++ {
		artworks, _, err := s.artworkRepo.List(ctx, models.ArtworkFilter{
			IsPublished: &published,
			Page:        page,
			PageSize:    pageSize,
		})
		if err != nil {
			return nil, fmt.Errorf("không lấy được danh sách tác phẩm cho sitemap: %w", err)
		}
		for _, a := range artworks {
			result = append(result, SitemapArtwork{ID: a.ID, UpdatedAt: a.UpdatedAt})
		}
		if len(artworks) < pageSize {
			break
		}
	}
	return result, nil
}

func (s *artworkService) SetFeatured(ctx context.Context, id int64, featured bool) error {
	return s.artworkRepo.SetFeatured(ctx, id, featured)
}

func (s *artworkService) SetFeaturedBatch(ctx context.Context, ids []int64, featured bool) error {
	if len(ids) == 0 {
		return fmt.Errorf("chưa chọn tác phẩm nào")
	}
	return s.artworkRepo.SetFeaturedBatch(ctx, ids, featured)
}

// enrichArtworks gộp thông tin học sinh/trường/khối lớp/giải/reaction/comment
// cho 1 danh sách artworks bằng batch query (tránh N+1) - dùng chung cho cả
// GetArtwork (1 item) và ListArtworks (nhiều item).
//
// Mọi tra cứu ở đây đều là truy vấn gộp: học sinh/giải/reaction/comment lấy
// theo lô id, trường và khối lớp đọc từ cache (17 dòng tĩnh). Tổng cộng cố
// định vài truy vấn cho mỗi trang, không phụ thuộc số tác phẩm trong trang.
//
// Học sinh thiếu (bản ghi bị xoá thủ công, dữ liệu cũ không nhất quán) chỉ
// làm trống tên tác giả chứ không làm hỏng cả trang - trước đây một id hỏng
// khiến toàn bộ danh sách trả lỗi 500.
// referenceData trả trường + khối lớp dạng map tra theo id, đọc từ cache khi
// còn hạn.
//
// Nếu hai request cùng gặp lúc cache hết hạn thì cả hai sẽ cùng nạp lại - chấp
// nhận được, vì đây chỉ là hai truy vấn nhỏ và mỗi 5 phút mới xảy ra một lần;
// đổi lại tránh được singleflight hay khoá ghi giữ suốt thời gian truy vấn.
func (s *artworkService) referenceData(ctx context.Context) (map[int64]*models.School, map[int64]*models.GradeLevel, map[int64]*models.TopicCategory, error) {
	s.refMu.RLock()
	if time.Now().Before(s.refExpiresAt) && s.cachedSchools != nil {
		schools, grades, topics := s.cachedSchools, s.cachedGrades, s.cachedTopicCategories
		s.refMu.RUnlock()
		return schools, grades, topics, nil
	}
	s.refMu.RUnlock()

	schools, err := s.schoolRepo.List(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load schools: %w", err)
	}
	schoolByID := make(map[int64]*models.School, len(schools))
	for _, sc := range schools {
		schoolByID[sc.ID] = sc
	}

	grades, err := s.gradeRepo.List(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load grade levels: %w", err)
	}
	gradeByID := make(map[int64]*models.GradeLevel, len(grades))
	for _, g := range grades {
		gradeByID[g.ID] = g
	}

	topics, err := s.topicCategoryRepo.List(ctx, true)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load topic categories: %w", err)
	}
	topicByID := make(map[int64]*models.TopicCategory, len(topics))
	for _, t := range topics {
		topicByID[t.ID] = t
	}

	s.refMu.Lock()
	s.cachedSchools = schoolByID
	s.cachedGrades = gradeByID
	s.cachedTopicCategories = topicByID
	s.refExpiresAt = time.Now().Add(refCacheTTL)
	s.refMu.Unlock()

	// Trả map vừa dựng chứ không đọc lại từ struct: giữa Unlock và lần đọc
	// tiếp theo có thể có goroutine khác đã ghi đè.
	return schoolByID, gradeByID, topicByID, nil
}

func (s *artworkService) enrichArtworks(ctx context.Context, artworks []*models.Artwork) ([]*models.ArtworkWithMeta, error) {
	if len(artworks) == 0 {
		return nil, nil
	}

	ids := make([]int64, len(artworks))
	studentIDs := make([]int64, len(artworks))
	for i, a := range artworks {
		ids[i] = a.ID
		studentIDs[i] = a.StudentID
	}

	studentByID, err := s.studentRepo.ListByIDs(ctx, studentIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to load students: %w", err)
	}

	awardsByArtwork, err := s.awardRepo.ListByArtworkIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to load awards: %w", err)
	}
	reactionsByArtwork, err := s.reactionRepo.CountByArtworkBatch(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to load reaction counts: %w", err)
	}
	commentsByArtwork, err := s.commentRepo.CountByArtworkBatch(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to load comment counts: %w", err)
	}

	schoolByID, gradeByID, topicByID, err := s.referenceData(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*models.ArtworkWithMeta, 0, len(artworks))
	for _, a := range artworks {
		student := studentByID[a.StudentID]

		meta := &models.ArtworkWithMeta{
			Artwork:        *a,
			CommentCount:   commentsByArtwork[a.ID],
			ReactionCounts: reactionsByArtwork[a.ID],
			Awards:         toAwardSlice(awardsByArtwork[a.ID]),
		}
		if student != nil {
			meta.StudentName = student.FullName
			meta.ClassName = student.ClassName
		}
		if sc, ok := schoolByID[a.SchoolID]; ok {
			meta.SchoolName = sc.Name
			meta.Region = sc.Region
		}
		if g, ok := gradeByID[a.GradeLevelID]; ok {
			meta.GradeLabel = g.Label
			meta.EducationLevel = g.EducationLevel
		}
		if a.TopicCategoryID != nil {
			if t, ok := topicByID[*a.TopicCategoryID]; ok {
				meta.TopicCategoryName = t.Name
			}
		}

		result = append(result, meta)
	}

	return result, nil
}

func toAwardSlice(awards []*models.Award) []models.Award {
	if len(awards) == 0 {
		return nil
	}
	out := make([]models.Award, len(awards))
	for i, a := range awards {
		out[i] = *a
	}
	return out
}
