package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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

// ActivityPoint là số liệu của 1 ngày trên biểu đồ xu hướng - dùng cho
// "nhịp hội thi 14 ngày gần nhất" ở Dashboard.
//
// Views đếm từ artwork_views (mỗi dòng = 1 lượt xem đã khử trùng lặp 24h),
// KHÔNG lấy từ artworks.view_count vì cột đó là tổng tích luỹ, không tách
// được theo ngày.
type ActivityPoint struct {
	Date      string `json:"date"` // YYYY-MM-DD
	Uploads   int64  `json:"uploads"`
	Views     int64  `json:"views"`
	Reactions int64  `json:"reactions"`
	Comments  int64  `json:"comments"`
}

// SchoolCoverage là mức độ tham gia của 1 trường: bao nhiêu khối lớp đã có
// bài trên tổng số khối. Đây là chỉ số hành động được - trường nào còn
// thiếu khối nào thì ban tổ chức biết phải nhắc ai.
type SchoolCoverage struct {
	SchoolID      int64  `json:"school_id"`
	Name          string `json:"name"`
	Region        string `json:"region"`
	Artworks      int64  `json:"artworks"`
	GradesCovered int64  `json:"grades_covered"`
	TotalGrades   int64  `json:"total_grades"`
	// Awarded là số tác phẩm của trường đã được trao giải.
	Awarded int64 `json:"awarded"`
}

// OperationsSnapshot gom các con số cần xử lý ngay (hàng chờ việc), tách
// khỏi các con số mô tả quy mô triển lãm.
type OperationsSnapshot struct {
	// PendingArtworks là tác phẩm đã upload nhưng chưa xuất bản.
	PendingArtworks int64 `json:"pending_artworks"`
	// HiddenComments là bình luận đã bị ẩn (đã xử lý) - dùng để đối chiếu
	// với tổng bình luận, cho biết mức độ spam.
	HiddenComments int64 `json:"hidden_comments"`
	TotalComments  int64 `json:"total_comments"`
	// AwardedArtworks / ActiveAwards cho biết tiến độ chấm giải.
	AwardedArtworks int64 `json:"awarded_artworks"`
	ActiveAwards    int64 `json:"active_awards"`
	// FeaturedArtworks là số tác phẩm đang được ghim nổi bật ở trang chủ.
	FeaturedArtworks int64 `json:"featured_artworks"`
	// SilentArtworks là tác phẩm đã xuất bản nhưng chưa có bất kỳ tương
	// tác nào (0 view, 0 reaction) - ứng viên cần đẩy lên trang chủ.
	SilentArtworks int64 `json:"silent_artworks"`
	// TotalViews / TotalReactions là tổng tương tác toàn triển lãm.
	TotalViews     int64 `json:"total_views"`
	TotalReactions int64 `json:"total_reactions"`
}

// RegionSummary là số tác phẩm + số học sinh của 1 khu vực - dùng cho dải
// card thống kê nhỏ ở đầu các trang quản trị Tác phẩm/Giải thưởng/Nhóm chủ
// đề. Tách khỏi DashboardStatsResponse (nặng hơn nhiều: activity,
// top_schools, top_artworks, school_coverage...) vì 3 trang đó chỉ cần đúng
// hai con số này.
type RegionSummary struct {
	Region   string `json:"region"`
	Artworks int64  `json:"artworks"`
	Students int64  `json:"students"`
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
	// ActivityTrend trả số liệu từng ngày trong khoảng [from, to] (cả hai đầu
	// đều tính theo lịch, giờ trong ngày bị bỏ qua). Nhận khoảng tường minh
	// thay vì "N ngày gần nhất" để phục vụ được cả bộ lọc theo tháng lẫn theo
	// khoảng ngày tuỳ chọn ở Dashboard.
	ActivityTrend(ctx context.Context, from, to time.Time) ([]ActivityPoint, error)
	// SchoolCoverageReport trả toàn bộ trường (kể cả trường 0 bài).
	SchoolCoverageReport(ctx context.Context) ([]SchoolCoverage, error)
	Operations(ctx context.Context) (OperationsSnapshot, error)
	// RegionSummaries trả số tác phẩm đã xuất bản + số học sinh của cả 3 khu
	// vực, theo thứ tự cố định saigon/cantho/vungtau.
	RegionSummaries(ctx context.Context) ([]RegionSummary, error)
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
	// Chỉ lấy cột dashboard thật sự dùng (ảnh hero + đối chiếu school_id).
	// Không SELECT cả artworkSelectColumns: thêm cột artwork mới (vd
	// topic_category_id) từng làm Scan lệch số đích và trắng cả trang /admin.
	query := `
		SELECT a.id, a.title, a.school_id, a.s3_url, a.thumbnail_url, a.variants, a.view_count,
		       st.full_name,
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
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top artworks by engagement: %w", err)
	}
	defer rows.Close()

	var results []ArtworkEngagement
	for rows.Next() {
		var e ArtworkEngagement
		var thumbnailURL sql.NullString

		err := rows.Scan(
			&e.ID, &e.Title, &e.SchoolID, &e.S3URL, &thumbnailURL, &e.Variants, &e.ViewCount,
			&e.StudentName, &e.ReactionCount, &e.CommentCount, &e.EngagementScore,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan artwork engagement: %w", err)
		}
		if thumbnailURL.Valid {
			e.ThumbnailURL = &thumbnailURL.String
		}
		results = append(results, e)
	}
	return results, rows.Err()
}

func (r *dashboardRepository) ActivityTrend(ctx context.Context, from, to time.Time) ([]ActivityPoint, error) {
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
	to = time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, to.Location())
	fromStr := from.Format("2006-01-02")
	toStr := to.Format("2006-01-02")

	// Bốn nguồn số liệu nằm ở bốn bảng khác nhau và không bảng nào chắc
	// chắn có dòng cho mọi ngày. Gom bằng UNION ALL rồi GROUP BY ngày thay
	// vì JOIN chéo: JOIN nhiều bảng "một-nhiều" theo ngày sẽ nhân bản dòng
	// và thổi phồng số đếm.
	query := `
		SELECT d AS day,
		       SUM(uploads) AS uploads,
		       SUM(views) AS views,
		       SUM(reactions) AS reactions,
		       SUM(comments) AS comments
		FROM (
			SELECT DATE(created_at) AS d, COUNT(*) AS uploads, 0 AS views, 0 AS reactions, 0 AS comments
			FROM artworks
			WHERE DATE(created_at) BETWEEN ? AND ?
			GROUP BY DATE(created_at)
			UNION ALL
			SELECT DATE(viewed_at), 0, COUNT(*), 0, 0
			FROM artwork_views
			WHERE DATE(viewed_at) BETWEEN ? AND ?
			GROUP BY DATE(viewed_at)
			UNION ALL
			SELECT DATE(created_at), 0, 0, COUNT(*), 0
			FROM artwork_reactions
			WHERE DATE(created_at) BETWEEN ? AND ?
			GROUP BY DATE(created_at)
			UNION ALL
			SELECT DATE(created_at), 0, 0, 0, COUNT(*)
			FROM artwork_comments
			WHERE DATE(created_at) BETWEEN ? AND ? AND is_hidden = 0
			GROUP BY DATE(created_at)
		) t
		GROUP BY d
		ORDER BY d
	`
	rows, err := r.db.QueryContext(ctx, query, fromStr, toStr, fromStr, toStr, fromStr, toStr, fromStr, toStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity trend: %w", err)
	}
	defer rows.Close()

	byDate := make(map[string]ActivityPoint)
	for rows.Next() {
		var p ActivityPoint
		if err := rows.Scan(&p.Date, &p.Uploads, &p.Views, &p.Reactions, &p.Comments); err != nil {
			return nil, fmt.Errorf("failed to scan activity point: %w", err)
		}
		// Driver có thể trả DATE dạng "2026-09-06 00:00:00" tuỳ parseTime -
		// cắt về đúng phần ngày để khớp khoá lịch bên dưới.
		if len(p.Date) > 10 {
			p.Date = p.Date[:10]
		}
		byDate[p.Date] = p
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Trả đủ điểm liên tục cho mọi ngày trong [from, to], kể cả ngày không
	// có hoạt động: biểu đồ đường mà thiếu ngày sẽ vẽ sai độ dốc (hai ngày
	// cách nhau một tuần bị nối thẳng như hai ngày liền kề). Cắt bớt phần
	// đầu/cuối không có dữ liệu là việc của tầng service (quyết định trình
	// bày), repository luôn trả đúng khoảng đã yêu cầu.
	totalDays := int(to.Sub(from).Hours()/24) + 1
	points := make([]ActivityPoint, 0, totalDays)
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		if p, ok := byDate[key]; ok {
			points = append(points, p)
			continue
		}
		points = append(points, ActivityPoint{Date: key})
	}
	return points, nil
}

func (r *dashboardRepository) SchoolCoverageReport(ctx context.Context) ([]SchoolCoverage, error) {
	// LEFT JOIN để trường chưa có tác phẩm nào vẫn xuất hiện với số 0 -
	// đó chính là trường ban tổ chức cần nhắc, bỏ khỏi báo cáo thì hỏng
	// mục đích của bảng này.
	query := `
		SELECT s.id, s.name, s.region,
		       COUNT(DISTINCT a.id) AS artworks,
		       COUNT(DISTINCT a.grade_level_id) AS grades_covered,
		       (SELECT COUNT(*) FROM grade_levels) AS total_grades,
		       COUNT(DISTINCT aa.artwork_id) AS awarded
		FROM schools s
		LEFT JOIN artworks a ON a.school_id = s.id AND a.is_published = 1
		LEFT JOIN artwork_awards aa ON aa.artwork_id = a.id
		WHERE s.is_active = 1
		GROUP BY s.id, s.name, s.region, s.display_order
		ORDER BY artworks DESC, s.display_order
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get school coverage: %w", err)
	}
	defer rows.Close()

	var results []SchoolCoverage
	for rows.Next() {
		var c SchoolCoverage
		if err := rows.Scan(&c.SchoolID, &c.Name, &c.Region, &c.Artworks,
			&c.GradesCovered, &c.TotalGrades, &c.Awarded); err != nil {
			return nil, fmt.Errorf("failed to scan school coverage: %w", err)
		}
		results = append(results, c)
	}
	return results, rows.Err()
}

func (r *dashboardRepository) Operations(ctx context.Context) (OperationsSnapshot, error) {
	var s OperationsSnapshot

	// Gom về một lần round-trip: mỗi số là một subquery vô hướng, rẻ hơn
	// nhiều so với 8 lần gọi DB riêng cho một trang tải mỗi lần mở.
	query := `
		SELECT
			(SELECT COUNT(*) FROM artworks WHERE is_published = 0),
			(SELECT COUNT(*) FROM artwork_comments WHERE is_hidden = 1),
			(SELECT COUNT(*) FROM artwork_comments),
			(SELECT COUNT(DISTINCT artwork_id) FROM artwork_awards),
			(SELECT COUNT(*) FROM awards WHERE is_active = 1),
			(SELECT COUNT(*) FROM artworks WHERE is_featured = 1 AND is_published = 1),
			(SELECT COUNT(*) FROM artworks a
			 WHERE a.is_published = 1 AND a.view_count = 0
			   AND NOT EXISTS (SELECT 1 FROM artwork_reactions r WHERE r.artwork_id = a.id)),
			(SELECT COALESCE(SUM(view_count), 0) FROM artworks WHERE is_published = 1),
			(SELECT COUNT(*) FROM artwork_reactions)
	`
	err := r.db.QueryRowContext(ctx, query).Scan(
		&s.PendingArtworks, &s.HiddenComments, &s.TotalComments,
		&s.AwardedArtworks, &s.ActiveAwards, &s.FeaturedArtworks,
		&s.SilentArtworks, &s.TotalViews, &s.TotalReactions,
	)
	if err != nil {
		return s, fmt.Errorf("failed to get operations snapshot: %w", err)
	}
	return s, nil
}

// countByRegion chạy 1 query "SELECT region, COUNT(*) ... GROUP BY region"
// và trả về map đã điền sẵn cả 3 khu vực = 0 - dùng chung cho cả hai nhánh
// đếm (tác phẩm/học sinh) của RegionSummaries để không lặp code Scan.
func (r *dashboardRepository) countByRegion(ctx context.Context, query string) (map[string]int64, error) {
	counts := map[string]int64{
		models.RegionSaigon:  0,
		models.RegionCanTho:  0,
		models.RegionVungTau: 0,
	}
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var region string
		var count int64
		if err := rows.Scan(&region, &count); err != nil {
			return nil, err
		}
		counts[region] = count
	}
	return counts, rows.Err()
}

func (r *dashboardRepository) RegionSummaries(ctx context.Context) ([]RegionSummary, error) {
	artworksByRegion, err := r.countByRegion(ctx, `
		SELECT s.region, COUNT(*)
		FROM artworks a
		JOIN schools s ON s.id = a.school_id
		WHERE a.is_published = 1
		GROUP BY s.region
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to count artworks by region: %w", err)
	}

	// Không lọc theo trạng thái nào: students không có cột đó, và theo quy
	// ước ở docs/detail_design/01-database.md, mỗi lần tạo tác phẩm luôn tạo
	// 1 bản ghi student mới - KHÔNG dedupe theo tên - nên đếm thẳng COUNT(*).
	studentsByRegion, err := r.countByRegion(ctx, `
		SELECT s.region, COUNT(*)
		FROM students st
		JOIN schools s ON s.id = st.school_id
		GROUP BY s.region
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to count students by region: %w", err)
	}

	// Thứ tự cố định để FE không phải tự sort.
	order := []string{models.RegionSaigon, models.RegionCanTho, models.RegionVungTau}
	result := make([]RegionSummary, 0, len(order))
	for _, region := range order {
		result = append(result, RegionSummary{
			Region:   region,
			Artworks: artworksByRegion[region],
			Students: studentsByRegion[region],
		})
	}
	return result, nil
}
