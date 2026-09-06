package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"s3-upload-tool/internal/repository"
)

// fakeDashboardRepo trả số liệu dựng sẵn, và ghi lại khoảng [from, to] mà
// service truyền xuống cho ActivityTrend.
type fakeDashboardRepo struct {
	activityFrom time.Time
	activityTo   time.Time
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

func (f *fakeDashboardRepo) ActivityTrend(_ context.Context, from, to time.Time) ([]repository.ActivityPoint, error) {
	f.activityFrom = from
	f.activityTo = to
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

	stats, err := NewDashboardService(repo).GetStats(context.Background(), time.Time{}, time.Time{})
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

// Không truyền from/to (cả hai zero-value, tức FE không lọc theo tháng/
// khoảng ngày) thì cửa sổ xu hướng phải đúng bằng activityTrendDays ngày gần
// nhất. Frontend cắt đôi chuỗi này để tính biến động 7 ngày, nên đổi con số
// ở service mà không có gì canh chừng sẽ làm phần trăm so sánh sai lệch âm
// thầm.
func TestGetStatsMacDinhDungCuaSoXuHuong(t *testing.T) {
	repo := &fakeDashboardRepo{}
	if _, err := NewDashboardService(repo).GetStats(context.Background(), time.Time{}, time.Time{}); err != nil {
		t.Fatalf("GetStats trả lỗi ngoài dự kiến: %v", err)
	}
	gotDays := int(repo.activityTo.Sub(repo.activityFrom).Hours()/24) + 1
	if gotDays != activityTrendDays {
		t.Errorf("ActivityTrend nhận khoảng %d ngày, mong đợi %d", gotDays, activityTrendDays)
	}
}

// Truyền from/to tường minh thì phải đi thẳng xuống ActivityTrend, không bị
// ép về mặc định 14 ngày - đây là đường cho bộ lọc "theo tháng"/"theo khoảng
// ngày" ở FE.
func TestGetStatsTruyenDungKhoangNgayTuyChinh(t *testing.T) {
	repo := &fakeDashboardRepo{}
	from := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)
	if _, err := NewDashboardService(repo).GetStats(context.Background(), from, to); err != nil {
		t.Fatalf("GetStats trả lỗi ngoài dự kiến: %v", err)
	}
	if !repo.activityFrom.Equal(from) || !repo.activityTo.Equal(to) {
		t.Errorf("ActivityTrend nhận [%v, %v], mong đợi [%v, %v]", repo.activityFrom, repo.activityTo, from, to)
	}
}

// Khoảng ngày quá dài (vd lọc nhầm) phải bị cắt về đúng maxActivityRangeDays,
// tính lùi từ `to` - chặn query UNION ALL quét toàn bộ 4 bảng qua nhiều năm.
func TestGetStatsChanKhoangNgayQuaDai(t *testing.T) {
	repo := &fakeDashboardRepo{}
	from := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC)
	if _, err := NewDashboardService(repo).GetStats(context.Background(), from, to); err != nil {
		t.Fatalf("GetStats trả lỗi ngoài dự kiến: %v", err)
	}
	gotDays := int(repo.activityTo.Sub(repo.activityFrom).Hours()/24) + 1
	if gotDays != maxActivityRangeDays {
		t.Errorf("ActivityTrend nhận khoảng %d ngày, mong đợi bị chặn về %d", gotDays, maxActivityRangeDays)
	}
	if !repo.activityTo.Equal(to) {
		t.Errorf("`to` phải giữ nguyên khi cắt khoảng, nhận %v", repo.activityTo)
	}
}

// Cắt điểm 0 hoạt động ở đầu/cuối chuỗi, nhưng GIỮ NGUYÊN điểm 0 xen giữa hai
// ngày có dữ liệu - đó là tín hiệu thật (ngày đó không ai thao tác gì), khác
// với phần đầu/cuối tháng chưa/không còn dữ liệu để đếm.
func TestGetStatsCatDauCuoiGiuNguyenXenGiua(t *testing.T) {
	repo := &fakeDashboardRepo{
		activity: []repository.ActivityPoint{
			{Date: "2026-08-01"}, // đầu tháng, chưa có gì - phải bị cắt
			{Date: "2026-08-02"}, // đầu tháng, chưa có gì - phải bị cắt
			{Date: "2026-08-03", Uploads: 2},
			{Date: "2026-08-04"}, // xen giữa - phải GIỮ NGUYÊN
			{Date: "2026-08-05", Views: 5},
			{Date: "2026-08-06"}, // cuối tháng, hết hoạt động - phải bị cắt
		},
	}
	stats, err := NewDashboardService(repo).GetStats(context.Background(), time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("GetStats trả lỗi ngoài dự kiến: %v", err)
	}

	wantDates := []string{"2026-08-03", "2026-08-04", "2026-08-05"}
	if len(stats.Activity) != len(wantDates) {
		t.Fatalf("Activity còn %d điểm, mong đợi %d: %+v", len(stats.Activity), len(wantDates), stats.Activity)
	}
	for i, want := range wantDates {
		if stats.Activity[i].Date != want {
			t.Errorf("điểm thứ %d có ngày %q, mong đợi %q", i, stats.Activity[i].Date, want)
		}
	}
}

// Lỗi ở bất kỳ nguồn nào cũng phải nổi lên thành lỗi của cả GetStats, kèm
// ngữ cảnh - Dashboard thà báo hỏng còn hơn hiển thị số 0 như thể đó là số
// liệu thật.
func TestGetStatsBaoLoiKhiMotNguonHong(t *testing.T) {
	for _, nguon := range []string{"total", "activity", "coverage", "ops"} {
		t.Run(nguon, func(t *testing.T) {
			repo := &fakeDashboardRepo{failOn: nguon}
			stats, err := NewDashboardService(repo).GetStats(context.Background(), time.Time{}, time.Time{})
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
