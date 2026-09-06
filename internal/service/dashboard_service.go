package service

import (
	"context"
	"fmt"

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

// activityTrendDays là bề rộng cửa sổ biểu đồ xu hướng. Hai tuần đủ để thấy
// hiệu ứng của một đợt phát động mà không làm trục hoành chật trên mobile.
const activityTrendDays = 14

// DashboardService gộp các query tổng hợp từ DashboardRepository thành 1
// response duy nhất cho trang Dashboard.
type DashboardService interface {
	GetStats(ctx context.Context) (*DashboardStatsResponse, error)
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

func (s *dashboardService) GetStats(ctx context.Context) (*DashboardStatsResponse, error) {
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

	activity, err := s.repo.ActivityTrend(ctx, activityTrendDays)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity trend: %w", err)
	}

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
