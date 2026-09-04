package repository

import (
	"context"
	"database/sql"
	"fmt"

	"s3-upload-tool/internal/database"
	"s3-upload-tool/internal/models"
)

// GradeLevelCount là số tác phẩm theo 1 khối lớp - dùng cho biểu đồ phân bổ
// theo khối lớp ở Dashboard.
type GradeLevelCount struct {
	GradeLevelID   int64  `json:"grade_level_id"`
	Label          string `json:"label"`
	EducationLevel string `json:"education_level"`
	Count          int64  `json:"count"`
}

// SchoolCount là số tác phẩm theo 1 trường - dùng cho "tác phẩm nhiều nhất"
// (top schools) ở Dashboard.
type SchoolCount struct {
	SchoolID int64  `json:"school_id"`
	Name     string `json:"name"`
	Region   string `json:"region"`
	Count    int64  `json:"count"`
}

// ArtworkEngagement là 1 tác phẩm kèm tổng lượt tương tác (view + reaction) -
// dùng cho "Top 10 tác phẩm hàng đầu" ở Dashboard. Title lấy qua embedding
// models.Artwork, không định nghĩa lại ở đây.
type ArtworkEngagement struct {
	models.Artwork
	StudentName     string `json:"student_name"`
	ReactionCount   int64  `json:"reaction_count"`
	CommentCount    int64  `json:"comment_count"`
	EngagementScore int64  `json:"engagement_score"`
}

// DashboardRepository chạy các query tổng hợp cho trang Dashboard - không
// map 1-1 với 1 bảng nào, luôn JOIN/GROUP BY qua artworks + schools/grade_levels.
type DashboardRepository interface {
	TotalArtworks(ctx context.Context) (int64, error)
	// TotalByRegion trả map region -> số tác phẩm (JOIN artworks-schools).
	TotalByRegion(ctx context.Context) (map[string]int64, error)
	TotalByGradeLevel(ctx context.Context) ([]GradeLevelCount, error)
	TopSchoolsByArtworkCount(ctx context.Context, limit int) ([]SchoolCount, error)
	// TopArtworksByEngagement sắp theo (view_count + số reaction) giảm dần.
	TopArtworksByEngagement(ctx context.Context, limit int) ([]ArtworkEngagement, error)
}

type dashboardRepository struct {
	db *database.DB
}

func NewDashboardRepository(db *database.DB) DashboardRepository {
	return &dashboardRepository{db: db}
}

func (r *dashboardRepository) TotalArtworks(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM artworks WHERE is_published = 1`).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to count total artworks: %w", err)
	}
	return total, nil
}

func (r *dashboardRepository) TotalByRegion(ctx context.Context) (map[string]int64, error) {
	query := `
		SELECT s.region, COUNT(*)
		FROM artworks a
		JOIN schools s ON s.id = a.school_id
		WHERE a.is_published = 1
		GROUP BY s.region
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to count by region: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int64)
	// Đảm bảo cả 3 khu vực luôn có key (0 nếu chưa có tác phẩm) để FE không
	// phải tự xử lý key thiếu khi vẽ chart.
	counts[models.RegionSaigon] = 0
	counts[models.RegionCanTho] = 0
	counts[models.RegionVungTau] = 0

	for rows.Next() {
		var region string
		var count int64
		if err := rows.Scan(&region, &count); err != nil {
			return nil, fmt.Errorf("failed to scan region count: %w", err)
		}
		counts[region] = count
	}
	return counts, rows.Err()
}

func (r *dashboardRepository) TotalByGradeLevel(ctx context.Context) ([]GradeLevelCount, error) {
	query := `
		SELECT g.id, g.label, g.education_level, COUNT(a.id)
		FROM grade_levels g
		LEFT JOIN artworks a ON a.grade_level_id = g.id AND a.is_published = 1
		GROUP BY g.id, g.label, g.education_level
		ORDER BY g.display_order
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to count by grade level: %w", err)
	}
	defer rows.Close()

	var results []GradeLevelCount
	for rows.Next() {
		var c GradeLevelCount
		if err := rows.Scan(&c.GradeLevelID, &c.Label, &c.EducationLevel, &c.Count); err != nil {
			return nil, fmt.Errorf("failed to scan grade level count: %w", err)
		}
		results = append(results, c)
	}
	return results, rows.Err()
}

func (r *dashboardRepository) TopSchoolsByArtworkCount(ctx context.Context, limit int) ([]SchoolCount, error) {
	if limit <= 0 {
		limit = 10
	}
	query := `
		SELECT s.id, s.name, s.region, COUNT(a.id) AS cnt
		FROM schools s
		LEFT JOIN artworks a ON a.school_id = s.id AND a.is_published = 1
		GROUP BY s.id, s.name, s.region
		ORDER BY cnt DESC
		LIMIT ?
	`
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top schools: %w", err)
	}
	defer rows.Close()

	var results []SchoolCount
	for rows.Next() {
		var c SchoolCount
		if err := rows.Scan(&c.SchoolID, &c.Name, &c.Region, &c.Count); err != nil {
			return nil, fmt.Errorf("failed to scan school count: %w", err)
		}
		results = append(results, c)
	}
	return results, rows.Err()
}

func (r *dashboardRepository) TopArtworksByEngagement(ctx context.Context, limit int) ([]ArtworkEngagement, error) {
	if limit <= 0 {
		limit = 10
	}
	query := fmt.Sprintf(`
		SELECT %s, st.full_name,
		       COALESCE(rc.reaction_count, 0) AS reaction_count,
		       COALESCE(cc.comment_count, 0) AS comment_count,
		       (a.view_count + COALESCE(rc.reaction_count, 0)) AS engagement_score
		FROM artworks a
		JOIN students st ON st.id = a.student_id
		LEFT JOIN (
			SELECT artwork_id, COUNT(*) AS reaction_count
			FROM artwork_reactions GROUP BY artwork_id
		) rc ON rc.artwork_id = a.id
		LEFT JOIN (
			SELECT artwork_id, COUNT(*) AS comment_count
			FROM artwork_comments WHERE is_hidden = 0 GROUP BY artwork_id
		) cc ON cc.artwork_id = a.id
		WHERE a.is_published = 1
		ORDER BY engagement_score DESC, a.created_at DESC
		LIMIT ?
	`, prefixColumns("a", artworkSelectColumns))

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top artworks by engagement: %w", err)
	}
	defer rows.Close()

	var results []ArtworkEngagement
	for rows.Next() {
		var e ArtworkEngagement
		var thumbnailURL sql.NullString
		var width, height sql.NullInt64
		var uploadID, createdBy sql.NullInt64

		err := rows.Scan(
			&e.ID, &e.Title, &e.StudentID, &e.SchoolID, &e.GradeLevelID, &e.S3Key, &e.S3URL, &thumbnailURL,
			&e.FileSize, &width, &height, &e.IsFeatured, &e.IsPublished, &e.ViewCount, &uploadID, &createdBy,
			&e.CreatedAt, &e.UpdatedAt,
			&e.StudentName, &e.ReactionCount, &e.CommentCount, &e.EngagementScore,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan artwork engagement: %w", err)
		}
		if thumbnailURL.Valid {
			e.ThumbnailURL = &thumbnailURL.String
		}
		if width.Valid {
			w := int(width.Int64)
			e.Width = &w
		}
		if height.Valid {
			h := int(height.Int64)
			e.Height = &h
		}
		if uploadID.Valid {
			e.UploadID = &uploadID.Int64
		}
		if createdBy.Valid {
			e.CreatedBy = &createdBy.Int64
		}
		results = append(results, e)
	}
	return results, rows.Err()
}
