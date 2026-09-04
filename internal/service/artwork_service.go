package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"

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
	Error    string `json:"error,omitempty"`
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
	Title        string
	StudentName  string
	SchoolID     int64
	GradeLevelID int64
	ClassName    string
	S3Key        string
	S3URL        string
	FileSize     int64
	Width        int
	Height       int
	AwardID      *int64
	CreatedBy    *int64
}

// UpdateArtworkRequest là input để cập nhật metadata artwork - KHÔNG đổi
// lại file S3 (đổi ảnh nghĩa là xoá tác phẩm cũ, tạo tác phẩm mới).
type UpdateArtworkRequest struct {
	Title        string
	StudentID    int64
	SchoolID     int64
	GradeLevelID int64
	IsFeatured   bool
	IsPublished  bool
	AwardID      *int64 // nil = không đổi gán giải; set rõ ID hoặc 0 để gỡ giải hiện tại
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
	// DeleteArtwork xoá DB record. Mặc định KHÔNG xoá S3 object (an toàn
	// hơn, tránh mất dữ liệu do bấm nhầm - dọn rác S3 định kỳ là việc ngoài
	// phạm vi service này, xem plan Phase 3 mục rủi ro #4).
	DeleteArtwork(ctx context.Context, id int64) error
	GetArtwork(ctx context.Context, id int64) (*models.ArtworkWithMeta, error)
	ListArtworks(ctx context.Context, filter models.ArtworkFilter) (*ArtworkListResult, error)
	SetFeatured(ctx context.Context, id int64, featured bool) error
}

type artworkService struct {
	db              *database.DB
	uploadService   UploadService
	artworkRepo     repository.ArtworkRepository
	studentRepo     repository.StudentRepository
	schoolRepo      repository.SchoolRepository
	gradeRepo       repository.GradeLevelRepository
	awardRepo       repository.AwardRepository
	reactionRepo    repository.ReactionRepository
	commentRepo     repository.CommentRepository
	bulkConcurrency int
}

func NewArtworkService(
	db *database.DB,
	uploadService UploadService,
	artworkRepo repository.ArtworkRepository,
	studentRepo repository.StudentRepository,
	schoolRepo repository.SchoolRepository,
	gradeRepo repository.GradeLevelRepository,
	awardRepo repository.AwardRepository,
	reactionRepo repository.ReactionRepository,
	commentRepo repository.CommentRepository,
) ArtworkService {
	return &artworkService{
		db:              db,
		uploadService:   uploadService,
		artworkRepo:     artworkRepo,
		studentRepo:     studentRepo,
		schoolRepo:      schoolRepo,
		gradeRepo:       gradeRepo,
		awardRepo:       awardRepo,
		reactionRepo:    reactionRepo,
		commentRepo:     commentRepo,
		bulkConcurrency: 5, // giới hạn upload song song, tránh áp đảo S3/mạng khi admin chọn hàng chục ảnh cùng lúc
	}
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
			results[idx] = item
		}(i, f)
	}

	wg.Wait()
	return results
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
		Title:        req.Title,
		StudentID:    student.ID,
		SchoolID:     req.SchoolID,
		GradeLevelID: req.GradeLevelID,
		S3Key:        req.S3Key,
		S3URL:        req.S3URL,
		FileSize:     req.FileSize,
		IsPublished:  true,
		CreatedBy:    req.CreatedBy,
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
	// bắt buộc phải cùng transaction với việc tạo artwork.
	if req.AwardID != nil {
		if err := s.awardRepo.AttachToArtwork(ctx, created.ID, *req.AwardID); err != nil {
			log.Printf("[ArtworkService] Tạo artwork thành công nhưng gán giải thất bại (id=%d, award=%d): %v", created.ID, *req.AwardID, err)
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
	existing.IsFeatured = req.IsFeatured
	existing.IsPublished = req.IsPublished

	if err := s.artworkRepo.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to update artwork: %w", err)
	}

	if req.AwardID != nil {
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
		if *req.AwardID > 0 {
			if err := s.awardRepo.AttachToArtwork(ctx, id, *req.AwardID); err != nil {
				log.Printf("[ArtworkService] Gán giải mới thất bại (artwork=%d, award=%d): %v", id, *req.AwardID, err)
			}
		}
	}

	return existing, nil
}

func (s *artworkService) DeleteArtwork(ctx context.Context, id int64) error {
	return s.artworkRepo.Delete(ctx, id)
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

func (s *artworkService) SetFeatured(ctx context.Context, id int64, featured bool) error {
	return s.artworkRepo.SetFeatured(ctx, id, featured)
}

// enrichArtworks gộp thông tin học sinh/trường/khối lớp/giải/reaction/comment
// cho 1 danh sách artworks bằng batch query (tránh N+1) - dùng chung cho cả
// GetArtwork (1 item) và ListArtworks (nhiều item).
func (s *artworkService) enrichArtworks(ctx context.Context, artworks []*models.Artwork) ([]*models.ArtworkWithMeta, error) {
	if len(artworks) == 0 {
		return nil, nil
	}

	ids := make([]int64, len(artworks))
	for i, a := range artworks {
		ids[i] = a.ID
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

	schools, err := s.schoolRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load schools: %w", err)
	}
	schoolByID := make(map[int64]*models.School, len(schools))
	for _, sc := range schools {
		schoolByID[sc.ID] = sc
	}

	grades, err := s.gradeRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load grade levels: %w", err)
	}
	gradeByID := make(map[int64]*models.GradeLevel, len(grades))
	for _, g := range grades {
		gradeByID[g.ID] = g
	}

	result := make([]*models.ArtworkWithMeta, 0, len(artworks))
	for _, a := range artworks {
		student, err := s.studentRepo.GetByID(ctx, a.StudentID)
		if err != nil {
			return nil, fmt.Errorf("failed to load student for artwork %d: %w", a.ID, err)
		}

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
