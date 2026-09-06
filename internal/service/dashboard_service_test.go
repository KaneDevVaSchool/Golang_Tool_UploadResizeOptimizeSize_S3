package service

import (
	"context"
	"errors"
	"testing"

	"s3-upload-tool/internal/repository"
)

// fakeDashboardRepo trả số liệu dựng sẵn, và ghi lại tham số `days` mà
// service truyền xuống cho ActivityTrend.
type fakeDashboardRepo struct {
	activityDays int
	failOn       string

	activity []repository.ActivityPoint
	coverage []repository.SchoolCoverage
	ops      repository.OperationsSnapshot
}

func (f *fakeDashboardRepo) TotalArtworks(context.Context) (int64, error) {
	if f.failOn == "total" {
		return 0, errors.New("lỗi giả lập")
	}
	return 42, nil
}

func (f *fakeDashboardRepo) TotalByRegion(context.Context) (map[string]int64, error) {
	return map[string]int64{"saigon": 30, "cantho": 8, "vungtau": 4}, nil
}

func (f *fakeDashboardRepo) TotalByGradeLevel(context.Context) ([]repository.GradeLevelCount, error) {
	return []repository.GradeLevelCount{
		{GradeLevelID: 1, Label: "Khối 1", EducationLevel: "primary", Count: 5},
	}, nil
}

func (f *fakeDashboardRepo) TopSchoolsByArtworkCount(context.Context, int) ([]repository.SchoolCount, error) {
	return []repository.SchoolCount{{SchoolID: 1, Name: "Phú Định", Region: "saigon", Count: 12}}, nil
}

func (f *fakeDashboardRepo) TopArtworksByEngagement(context.Context, int) ([]repository.ArtworkEngagement, error) {
	return nil, nil
}

func (f *fakeDashboardRepo) ActivityTrend(_ context.Context, days int) ([]repository.ActivityPoint, error) {
	f.activityDays = days
	if f.failOn == "activity" {
		return nil, errors.New("lỗi giả lập")
	}
	return f.activity, nil
}

func (f *fakeDashboardRepo) SchoolCoverageReport(context.Context) ([]repository.SchoolCoverage, error) {
	if f.failOn == "coverage" {
		return nil, errors.New("lỗi giả lập")
	}
	return f.coverage, nil
}

func (f *fakeDashboardRepo) Operations(context.Context) (repository.OperationsSnapshot, error) {
	if f.failOn == "ops" {
		return repository.OperationsSnapshot{}, errors.New("lỗi giả lập")
	}
	return f.ops, nil
}

func (f *fakeDashboardRepo) RegionSummaries(context.Context) ([]repository.RegionSummary, error) {
	if f.failOn == "region_summary" {
		return nil, errors.New("lỗi giả lập")
	}
	return []repository.RegionSummary{
		{Region: "saigon", Artworks: 30, Students: 35},
		{Region: "cantho", Artworks: 8, Students: 9},
		{Region: "vungtau", Artworks: 4, Students: 4},
	}, nil
}

// GetStats phải gộp đủ CẢ BẢY nguồn số liệu vào một response. Bài test khoá
// lại điều này vì thêm một nguồn mới mà quên nối vào DTO là lỗi im lặng:
// build vẫn xanh, frontend chỉ thấy trường rỗng.
func TestGetStatsGomDuMoiNguonSoLieu(t *testing.T) {
	repo := &fakeDashboardRepo{
		activity: []repository.ActivityPoint{{Date: "2026-09-01", Uploads: 3, Views: 10}},
		coverage: []repository.SchoolCoverage{
			{SchoolID: 1, Name: "Phú Định", Region: "saigon", Artworks: 12, GradesCovered: 9, TotalGrades: 12},
		},
		ops: repository.OperationsSnapshot{PendingArtworks: 4, TotalViews: 900, ActiveAwards: 3},
	}

	stats, err := NewDashboardService(repo).GetStats(context.Background())
	if err != nil {
		t.Fatalf("GetStats trả lỗi ngoài dự kiến: %v", err)
	}

	if stats.TotalArtworks != 42 {
		t.Errorf("TotalArtworks = %d, mong đợi 42", stats.TotalArtworks)
	}
	if len(stats.Activity) != 1 || stats.Activity[0].Uploads != 3 {
		t.Errorf("Activity không được nối vào response: %+v", stats.Activity)
	}
	if len(stats.SchoolCoverage) != 1 || stats.SchoolCoverage[0].Name != "Phú Định" {
		t.Errorf("SchoolCoverage không được nối vào response: %+v", stats.SchoolCoverage)
	}
	if stats.Operations.PendingArtworks != 4 || stats.Operations.TotalViews != 900 {
		t.Errorf("Operations không được nối vào response: %+v", stats.Operations)
	}
	if stats.TotalByRegion["saigon"] != 30 {
		t.Errorf("TotalByRegion sai: %+v", stats.TotalByRegion)
	}
}

// Cửa sổ xu hướng phải đúng bằng activityTrendDays. Frontend cắt đôi chuỗi
// này để tính biến động 7 ngày, nên đổi con số ở service mà không có gì
// canh chừng sẽ làm phần trăm so sánh sai lệch âm thầm.
func TestGetStatsDungDungCuaSoXuHuong(t *testing.T) {
	repo := &fakeDashboardRepo{}
	if _, err := NewDashboardService(repo).GetStats(context.Background()); err != nil {
		t.Fatalf("GetStats trả lỗi ngoài dự kiến: %v", err)
	}
	if repo.activityDays != activityTrendDays {
		t.Errorf("ActivityTrend nhận days = %d, mong đợi %d", repo.activityDays, activityTrendDays)
	}
}

// Lỗi ở bất kỳ nguồn nào cũng phải nổi lên thành lỗi của cả GetStats, kèm
// ngữ cảnh - Dashboard thà báo hỏng còn hơn hiển thị số 0 như thể đó là số
// liệu thật.
func TestGetStatsBaoLoiKhiMotNguonHong(t *testing.T) {
	for _, nguon := range []string{"total", "activity", "coverage", "ops"} {
		t.Run(nguon, func(t *testing.T) {
			repo := &fakeDashboardRepo{failOn: nguon}
			stats, err := NewDashboardService(repo).GetStats(context.Background())
			if err == nil {
				t.Fatalf("nguồn %q hỏng nhưng GetStats vẫn trả thành công", nguon)
			}
			if stats != nil {
				t.Errorf("có lỗi thì phải trả nil, nhận %+v", stats)
			}
		})
	}
}

// GetRegionSummary phải trả đúng 3 khu vực theo thứ tự cố định
// saigon/cantho/vungtau - dải card ở 3 trang quản trị dựa vào thứ tự này để
// không phải tự sort ở frontend.
func TestGetRegionSummaryTraDungThuTuVaSoLieu(t *testing.T) {
	repo := &fakeDashboardRepo{}
	summary, err := NewDashboardService(repo).GetRegionSummary(context.Background())
	if err != nil {
		t.Fatalf("GetRegionSummary trả lỗi ngoài dự kiến: %v", err)
	}

	want := []repository.RegionSummary{
		{Region: "saigon", Artworks: 30, Students: 35},
		{Region: "cantho", Artworks: 8, Students: 9},
		{Region: "vungtau", Artworks: 4, Students: 4},
	}
	if len(summary) != len(want) {
		t.Fatalf("GetRegionSummary trả %d khu vực, mong đợi %d", len(summary), len(want))
	}
	for i, item := range want {
		if summary[i] != item {
			t.Errorf("khu vực thứ %d = %+v, mong đợi %+v", i, summary[i], item)
		}
	}
}

// Lỗi từ repository phải nổi lên thành lỗi của GetRegionSummary, kèm ngữ
// cảnh - cùng nguyên tắc với GetStats: thà báo hỏng còn hơn hiển thị số 0
// như thể đó là số liệu thật.
func TestGetRegionSummaryBaoLoiKhiRepoHong(t *testing.T) {
	repo := &fakeDashboardRepo{failOn: "region_summary"}
	summary, err := NewDashboardService(repo).GetRegionSummary(context.Background())
	if err == nil {
		t.Fatal("repo hỏng nhưng GetRegionSummary vẫn trả thành công")
	}
	if summary != nil {
		t.Errorf("có lỗi thì phải trả nil, nhận %+v", summary)
	}
}
