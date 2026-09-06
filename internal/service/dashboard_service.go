package service

import (
	"context"
	"fmt"
	"time"

	"s3-upload-tool/internal/repository"
)

// DashboardStatsResponse gộp toàn bộ số liệu Dashboard thành 1 DTO trả về
// cho FE trong 1 lần gọi API - tránh FE phải gọi nhiều endpoint riêng lẻ.
type DashboardStatsResponse struct {
	TotalArtworks int64                          `json:"total_artworks"`
	TotalByRegion map[string]int64               `json:"total_by_region"`
	TotalByGrade  []repository.GradeLevelCount   `json:"total_by_grade"`
	TopSchools    []repository.SchoolCount       `json:"top_schools"`
	TopArtworks   []repository.ArtworkEngagement `json:"top_artworks"`
	// Activity là nhịp hoạt động từng ngày (mặc định 14 ngày gần nhất).
	Activity []repository.ActivityPoint `json:"activity"`
	// SchoolCoverage cho biết trường nào còn thiếu bài / thiếu khối.
	SchoolCoverage []repository.SchoolCoverage `json:"school_coverage"`
	// Operations là các con số cần hành động (hàng chờ duyệt, chấm giải...).
	Operations repository.OperationsSnapshot `json:"operations"`
}

// activityTrendDays là bề rộng mặc định của biểu đồ xu hướng khi không có bộ
// lọc theo tháng/khoảng ngày nào được chỉ định. Hai tuần đủ để thấy hiệu ứng
// của một đợt phát động mà không làm trục hoành chật trên mobile.
const activityTrendDays = 14

// maxActivityRangeDays chặn khoảng ngày quá dài (vd chọn nhầm from=2000-01-01)
// làm query UNION ALL quét toàn bộ 4 bảng - đây là endpoint admin, không có
// rate limit riêng như public, nên phải tự chặn ở đây.
const maxActivityRangeDays = 366

// DashboardService gộp các query tổng hợp từ DashboardRepository thành 1
// response duy nhất cho trang Dashboard.
type DashboardService interface {
	// GetStats trả toàn bộ số liệu Dashboard. from/to là khoảng ngày cho biểu
	// đồ nhịp hoạt động (cả hai đầu cùng zero-value time.Time) thì dùng mặc
	// định activityTrendDays ngày gần nhất - giữ hành vi cũ khi FE không gửi
	// bộ lọc theo tháng/khoảng ngày.
	GetStats(ctx context.Context, from, to time.Time) (*DashboardStatsResponse, error)
	// GetRegionSummary trả số tác phẩm + số học sinh mỗi khu vực - phiên bản
	// nhẹ của GetStats, dùng cho dải card đầu trang Tác phẩm/Giải thưởng/Nhóm
	// chủ đề (không cần activity/top_schools/operations...).
	GetRegionSummary(ctx context.Context) ([]repository.RegionSummary, error)
}

type dashboardService struct {
	repo repository.DashboardRepository
}

func NewDashboardService(repo repository.DashboardRepository) DashboardService {
	return &dashboardService{repo: repo}
}

func (s *dashboardService) GetStats(ctx context.Context, from, to time.Time) (*DashboardStatsResponse, error) {
	total, err := s.repo.TotalArtworks(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get total artworks: %w", err)
	}

	byRegion, err := s.repo.TotalByRegion(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get total by region: %w", err)
	}

	byGrade, err := s.repo.TotalByGradeLevel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get total by grade level: %w", err)
	}

	topSchools, err := s.repo.TopSchoolsByArtworkCount(ctx, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get top schools: %w", err)
	}

	topArtworks, err := s.repo.TopArtworksByEngagement(ctx, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to get top artworks: %w", err)
	}

	from, to = resolveActivityRange(from, to)
	activity, err := s.repo.ActivityTrend(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity trend: %w", err)
	}
	activity = trimLeadingTrailingZero(activity)

	coverage, err := s.repo.SchoolCoverageReport(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get school coverage: %w", err)
	}

	ops, err := s.repo.Operations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get operations snapshot: %w", err)
	}

	return &DashboardStatsResponse{
		TotalArtworks:  total,
		TotalByRegion:  byRegion,
		TotalByGrade:   byGrade,
		TopSchools:     topSchools,
		TopArtworks:    topArtworks,
		Activity:       activity,
		SchoolCoverage: coverage,
		Operations:     ops,
	}, nil
}

func (s *dashboardService) GetRegionSummary(ctx context.Context) ([]repository.RegionSummary, error) {
	summary, err := s.repo.RegionSummaries(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get region summary: %w", err)
	}
	return summary, nil
}

// resolveActivityRange chuẩn hoá khoảng ngày cho biểu đồ nhịp hoạt động.
// from/to cùng là zero-value (handler không nhận được from/to hợp lệ từ
// query, hoặc gọi nội bộ không truyền) thì dùng mặc định activityTrendDays
// ngày gần nhất - giữ đúng hành vi trước khi có bộ lọc theo tháng/khoảng
// ngày. Khoảng dài hơn maxActivityRangeDays bị cắt về đúng giới hạn đó, tính
// từ `to` lùi lại, để tránh query quét quá nhiều dữ liệu.
func resolveActivityRange(from, to time.Time) (time.Time, time.Time) {
	if from.IsZero() && to.IsZero() {
		now := time.Now()
		return now.AddDate(0, 0, -(activityTrendDays - 1)), now
	}
	if to.IsZero() {
		to = time.Now()
	}
	if from.IsZero() || from.After(to) {
		from = to.AddDate(0, 0, -(activityTrendDays - 1))
	}
	if to.Sub(from).Hours()/24 > maxActivityRangeDays-1 {
		from = to.AddDate(0, 0, -(maxActivityRangeDays - 1))
	}
	return from, to
}

// trimLeadingTrailingZero cắt bỏ các điểm 0 hoạt động ở đầu và cuối chuỗi -
// tháng vừa chọn còn dở dang (vd hôm nay là ngày 7, còn 23 ngày chưa tới thì
// chưa thể có hoạt động) sẽ vẽ một đoạn thẳng nằm ngang vô nghĩa nếu giữ
// nguyên. Ngày 0 hoạt động nằm XEN GIỮA hai ngày có dữ liệu vẫn được giữ
// nguyên - đó là tín hiệu thật (hôm đó không ai thao tác gì), khác với phần
// đầu/cuối chưa/không còn dữ liệu để đếm.
func trimLeadingTrailingZero(points []repository.ActivityPoint) []repository.ActivityPoint {
	isZero := func(p repository.ActivityPoint) bool {
		return p.Uploads == 0 && p.Views == 0 && p.Reactions == 0 && p.Comments == 0
	}

	start := 0
	for start < len(points) && isZero(points[start]) {
		start++
	}
	if start == len(points) {
		// Toàn bộ khoảng không có hoạt động - giữ nguyên chuỗi rỗng-hoá thay
		// vì cắt sạch, để FE còn phân biệt được "không có dữ liệu" với "chưa
		// tải xong".
		return points
	}

	end := len(points) - 1
	for end > start && isZero(points[end]) {
		end--
	}
	return points[start : end+1]
}
