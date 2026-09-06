package service

import (
	"context"
	"errors"
	"testing"

	"s3-upload-tool/internal/models"
)

// fakeArtworkDownloadRepo ghi lại lời gọi Create - chỉ phương thức duy nhất
// của ArtworkDownloadRepository nên không cần helper "không dùng trong test
// này" như fakeArtworkRepo.
type fakeArtworkDownloadRepo struct {
	created []*models.ArtworkDownload
	err     error
}

func (f *fakeArtworkDownloadRepo) Create(_ context.Context, download *models.ArtworkDownload) error {
	if f.err != nil {
		return f.err
	}
	f.created = append(f.created, download)
	return nil
}

// LogDownload phải chuyển thẳng dữ liệu xuống repository, không sửa đổi gì -
// việc "lỗi ghi log không chặn tải" là trách nhiệm của handler (xem
// logArtworkDownload trong internal/handlers), không phải của service.
func TestLogDownloadChuyenDungDuLieuXuongRepo(t *testing.T) {
	repo := &fakeArtworkDownloadRepo{}
	svc := &artworkService{downloadRepo: repo}

	adminID := int64(7)
	input := &models.ArtworkDownload{
		ArtworkID:   42,
		AdminUserID: &adminID,
		Source:      models.ArtworkDownloadSourceAdmin,
		IPAddress:   "127.0.0.1",
		UserAgent:   "go-test",
	}

	if err := svc.LogDownload(context.Background(), input); err != nil {
		t.Fatalf("không muốn lỗi, nhận %v", err)
	}
	if len(repo.created) != 1 {
		t.Fatalf("muốn Create được gọi đúng 1 lần, nhận %d", len(repo.created))
	}
	got := repo.created[0]
	if got.ArtworkID != 42 || got.Source != models.ArtworkDownloadSourceAdmin {
		t.Errorf("dữ liệu truyền xuống repo không khớp input: %+v", got)
	}
	if got.AdminUserID == nil || *got.AdminUserID != 7 {
		t.Errorf("muốn AdminUserID=7, nhận %v", got.AdminUserID)
	}
}

// Lỗi từ repository phải được trả nguyên lên trên, không bị nuốt ở service.
func TestLogDownloadLanTruyenLoiTuRepo(t *testing.T) {
	wantErr := errors.New("lỗi ghi DB giả lập")
	repo := &fakeArtworkDownloadRepo{err: wantErr}
	svc := &artworkService{downloadRepo: repo}

	err := svc.LogDownload(context.Background(), &models.ArtworkDownload{ArtworkID: 1})
	if !errors.Is(err, wantErr) {
		t.Fatalf("muốn lỗi %v được lan truyền, nhận %v", wantErr, err)
	}
}

// Chưa nối dây downloadRepo (vd DB tắt) phải báo lỗi rõ ràng thay vì panic
// nil pointer.
func TestLogDownloadThieuRepoTraLoiRoRang(t *testing.T) {
	svc := &artworkService{}
	if err := svc.LogDownload(context.Background(), &models.ArtworkDownload{ArtworkID: 1}); err == nil {
		t.Fatal("muốn lỗi khi downloadRepo chưa cấu hình, nhận nil")
	}
}
