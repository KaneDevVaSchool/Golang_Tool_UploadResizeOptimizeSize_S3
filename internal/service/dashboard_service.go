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
}

// DashboardService gộp các query tổng hợp từ DashboardRepository thành 1
// response duy nhất cho trang Dashboard.
type DashboardService interface {
	GetStats(ctx context.Context) (*DashboardStatsResponse, error)
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

	return &DashboardStatsResponse{
		TotalArtworks: total,
		TotalByRegion: byRegion,
		TotalByGrade:  byGrade,
		TopSchools:    topSchools,
		TopArtworks:   topArtworks,
	}, nil
}
