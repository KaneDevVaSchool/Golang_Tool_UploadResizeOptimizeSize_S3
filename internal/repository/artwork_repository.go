package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"s3-upload-tool/internal/database"
	"s3-upload-tool/internal/models"
)

// ArtworkRepository quản lý bảng trung tâm artworks - CRUD + list có
// filter/search/pagination cho cả trang quản trị lẫn API public.
type ArtworkRepository interface {
	// Create nhận *database.Tx vì luôn tạo cùng transaction với students
	// (ArtworkService.CreateArtworkFromUpload) - theo đúng pattern
	// UploadRepository.CreateUpload.
	Create(ctx context.Context, tx *database.Tx, artwork *models.Artwork) (*models.Artwork, error)
	Update(ctx context.Context, artwork *models.Artwork) error
	Delete(ctx context.Context, id int64) error
	// DeleteBatch xoá nhiều bản ghi trong 1 câu DELETE ... WHERE id IN (...) -
	// dùng cho thao tác xoá hàng loạt ở trang quản trị. Trả về số dòng thực sự
	// bị xoá (có thể nhỏ hơn len(ids) nếu vài id không tồn tại - không coi là
	// lỗi, giống SetFeaturedBatch).
	DeleteBatch(ctx context.Context, ids []int64) (int64, error)
	GetByID(ctx context.Context, id int64) (*models.Artwork, error)
	// List trả (items, totalCount, error) - totalCount phục vụ pagination
	// UI (tổng số trang), tính bằng query COUNT(*) riêng cùng điều kiện WHERE.
	List(ctx context.Context, filter models.ArtworkFilter) ([]*models.Artwork, int64, error)
	SetFeatured(ctx context.Context, id int64, featured bool) error
	// SetFeaturedBatch cập nhật is_featured cho nhiều id trong 1 câu UPDATE -
	// dùng cho thao tác bulk ở trang quản trị, tránh N round-trip khi admin
	// chọn hàng chục tác phẩm cùng lúc. Id không tồn tại bị bỏ qua lặng lẽ
	// (không coi là lỗi) vì RowsAffected < len(ids) không tự nó là bất thường.
	SetFeaturedBatch(ctx context.Context, ids []int64, featured bool) error
}

type artworkRepository struct {
	db *database.DB
}

func NewArtworkRepository(db *database.DB) ArtworkRepository {
	return &artworkRepository{db: db}
}

const artworkSelectColumns = `id, title, student_id, school_id, grade_level_id, topic_category_id, s3_key, s3_url, thumbnail_url, variants, file_size, width, height, is_featured, is_published, view_count, upload_id, created_by, created_at, updated_at`

func scanArtwork(scanner interface{ Scan(dest ...any) error }) (*models.Artwork, error) {
	a := &models.Artwork{}
	var thumbnailURL sql.NullString
	var width, height sql.NullInt64
	var uploadID, createdBy sql.NullInt64
	var topicCategoryID sql.NullInt64

	err := scanner.Scan(
		&a.ID, &a.Title, &a.StudentID, &a.SchoolID, &a.GradeLevelID, &topicCategoryID, &a.S3Key, &a.S3URL, &thumbnailURL,
		&a.Variants,
		&a.FileSize, &width, &height, &a.IsFeatured, &a.IsPublished, &a.ViewCount, &uploadID, &createdBy,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if thumbnailURL.Valid {
		a.ThumbnailURL = &thumbnailURL.String
	}
	if width.Valid {
		w := int(width.Int64)
		a.Width = &w
	}
	if height.Valid {
		h := int(height.Int64)
		a.Height = &h
	}
	if uploadID.Valid {
		a.UploadID = &uploadID.Int64
	}
	if createdBy.Valid {
		a.CreatedBy = &createdBy.Int64
	}
	if topicCategoryID.Valid {
		a.TopicCategoryID = &topicCategoryID.Int64
	}
	return a, nil
}

func (r *artworkRepository) Create(ctx context.Context, tx *database.Tx, artwork *models.Artwork) (*models.Artwork, error) {
	now := time.Now()
	query := `
		INSERT INTO artworks (
			title, student_id, school_id, grade_level_id, topic_category_id, s3_key, s3_url, thumbnail_url, variants,
			file_size, width, height, is_featured, is_published, view_count, upload_id, created_by,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?)
	`
	result, err := tx.ExecContext(ctx, query,
		artwork.Title, artwork.StudentID, artwork.SchoolID, artwork.GradeLevelID, artwork.TopicCategoryID, artwork.S3Key, artwork.S3URL, artwork.ThumbnailURL, artwork.Variants,
		artwork.FileSize, artwork.Width, artwork.Height, artwork.IsFeatured, artwork.IsPublished, artwork.UploadID, artwork.CreatedBy,
		now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create artwork: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	artwork.ID = id
	artwork.ViewCount = 0
	artwork.CreatedAt = now
	artwork.UpdatedAt = now
	return artwork, nil
}

func (r *artworkRepository) Update(ctx context.Context, artwork *models.Artwork) error {
	query := `
		UPDATE artworks
		SET title = ?, student_id = ?, school_id = ?, grade_level_id = ?, topic_category_id = ?,
		    is_featured = ?, is_published = ?, updated_at = ?
		WHERE id = ?
	`
	now := time.Now()
	result, err := r.db.ExecContext(ctx, query,
		artwork.Title, artwork.StudentID, artwork.SchoolID, artwork.GradeLevelID, artwork.TopicCategoryID,
		artwork.IsFeatured, artwork.IsPublished, now, artwork.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update artwork: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("artwork not found: id=%d", artwork.ID)
	}
	artwork.UpdatedAt = now
	return nil
}

func (r *artworkRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM artworks WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete artwork: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("artwork not found: id=%d", id)
	}
	return nil
}

func (r *artworkRepository) DeleteBatch(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`DELETE FROM artworks WHERE id IN (%s)`, strings.Join(placeholders, ","))
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to delete artworks batch: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	return rows, nil
}

func (r *artworkRepository) GetByID(ctx context.Context, id int64) (*models.Artwork, error) {
	query := fmt.Sprintf(`SELECT %s FROM artworks WHERE id = ?`, artworkSelectColumns)
	a, err := scanArtwork(r.db.QueryRowContext(ctx, query, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get artwork by id: %w", err)
	}
	return a, nil
}

func (r *artworkRepository) SetFeatured(ctx context.Context, id int64, featured bool) error {
	result, err := r.db.ExecContext(ctx, `UPDATE artworks SET is_featured = ?, updated_at = ? WHERE id = ?`, featured, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to set featured: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("artwork not found: id=%d", id)
	}
	return nil
}

func (r *artworkRepository) SetFeaturedBatch(ctx context.Context, ids []int64, featured bool) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, 0, len(ids)+2)
	args = append(args, featured, time.Now())
	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}

	query := fmt.Sprintf(
		`UPDATE artworks SET is_featured = ?, updated_at = ? WHERE id IN (%s)`,
		strings.Join(placeholders, ","),
	)
	if _, err := r.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to set featured batch: %w", err)
	}
	return nil
}

// List xây WHERE động theo filter, dùng LIKE cho search (an toàn với dấu
// tiếng Việt hơn FULLTEXT - xem docs/plan phase 3, mục rủi ro #2: FULLTEXT
// tokenizer tiếng Việt có dấu có thể không khớp chính xác, LIKE chấp nhận
// chậm hơn nhưng đúng với quy mô dữ liệu hội thi vẽ tranh).
func (r *artworkRepository) List(ctx context.Context, filter models.ArtworkFilter) ([]*models.Artwork, int64, error) {
	var where []string
	var args []any

	// search khớp cả title (artworks) và tên học sinh (students) - cần JOIN
	// students để lọc theo tên, nhưng SELECT chính vẫn chỉ trả cột artworks
	// (giữ scanArtwork dùng chung cho mọi nhánh).
	joinStudents := filter.Search != ""

	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		where = append(where, "(artworks.title LIKE ? OR students.full_name LIKE ?)")
		args = append(args, like, like)
	}
	if filter.SchoolID != nil {
		where = append(where, "artworks.school_id = ?")
		args = append(args, *filter.SchoolID)
	}
	if filter.Region != nil && *filter.Region != "" {
		where = append(where, "artworks.school_id IN (SELECT id FROM schools WHERE region = ?)")
		args = append(args, *filter.Region)
	}
	if filter.GradeLevelID != nil {
		where = append(where, "artworks.grade_level_id = ?")
		args = append(args, *filter.GradeLevelID)
	}
	if filter.EducationLevel != "" {
		where = append(where, "artworks.grade_level_id IN (SELECT id FROM grade_levels WHERE education_level = ?)")
		args = append(args, filter.EducationLevel)
	}
	if filter.TopicCategoryID != nil {
		where = append(where, "artworks.topic_category_id = ?")
		args = append(args, *filter.TopicCategoryID)
	}
	if filter.AwardID != nil {
		where = append(where, "artworks.id IN (SELECT artwork_id FROM artwork_awards WHERE award_id = ?)")
		args = append(args, *filter.AwardID)
	}
	if filter.HasAward != nil {
		if *filter.HasAward {
			where = append(where, "EXISTS (SELECT 1 FROM artwork_awards aa WHERE aa.artwork_id = artworks.id)")
		} else {
			where = append(where, "NOT EXISTS (SELECT 1 FROM artwork_awards aa WHERE aa.artwork_id = artworks.id)")
		}
	}
	if filter.IsFeatured != nil {
		where = append(where, "artworks.is_featured = ?")
		args = append(args, *filter.IsFeatured)
	}
	if filter.IsPublished != nil {
		where = append(where, "artworks.is_published = ?")
		args = append(args, *filter.IsPublished)
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	fromClause := "FROM artworks"
	if joinStudents {
		fromClause += " JOIN students ON students.id = artworks.student_id"
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(*) %s %s`, fromClause, whereClause)
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count artworks: %w", err)
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	listQuery := fmt.Sprintf(
		`SELECT %s %s %s ORDER BY artworks.created_at DESC LIMIT ? OFFSET ?`,
		prefixColumns("artworks", artworkSelectColumns), fromClause, whereClause,
	)
	listArgs := append(append([]any{}, args...), pageSize, offset)

	rows, err := r.db.QueryContext(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list artworks: %w", err)
	}
	defer rows.Close()

	var artworks []*models.Artwork
	for rows.Next() {
		a, err := scanArtwork(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan artwork: %w", err)
		}
		artworks = append(artworks, a)
	}
	return artworks, total, rows.Err()
}
