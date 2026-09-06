package service

import (
	"context"
	"errors"
	"testing"

	"s3-upload-tool/internal/database"
	"s3-upload-tool/internal/models"
)

// fakeArtworkRepo chỉ ghi lại lời gọi SetFeaturedBatch - các phương thức khác
// của ArtworkRepository không cần thiết cho bài test bulk-featured nên trả
// lỗi rõ ràng nếu lỡ bị gọi tới, để test khác không âm thầm dùng nhầm fake này.
type fakeArtworkRepo struct {
	batchIDs   []int64
	batchValue bool
	batchCalls int
	batchErr   error
}

func (f *fakeArtworkRepo) Create(context.Context, *database.Tx, *models.Artwork) (*models.Artwork, error) {
	return nil, errors.New("không dùng trong test này")
}
func (f *fakeArtworkRepo) Update(context.Context, *models.Artwork) error {
	return errors.New("không dùng trong test này")
}
func (f *fakeArtworkRepo) Delete(context.Context, int64) error {
	return errors.New("không dùng trong test này")
}
func (f *fakeArtworkRepo) GetByID(context.Context, int64) (*models.Artwork, error) {
	return nil, errors.New("không dùng trong test này")
}
func (f *fakeArtworkRepo) List(context.Context, models.ArtworkFilter) ([]*models.Artwork, int64, error) {
	return nil, 0, errors.New("không dùng trong test này")
}
func (f *fakeArtworkRepo) SetFeatured(context.Context, int64, bool) error {
	return errors.New("không dùng trong test này")
}
func (f *fakeArtworkRepo) SetFeaturedBatch(_ context.Context, ids []int64, featured bool) error {
	f.batchCalls++
	f.batchIDs = ids
	f.batchValue = featured
	return f.batchErr
}

// SetFeaturedBatch phải từ chối danh sách rỗng ở tầng service (không chạm
// repository) - tránh chạy một UPDATE ... WHERE id IN () vô nghĩa nếu frontend
// có lỗi gửi mảng rỗng.
func TestSetFeaturedBatchTuChoiDanhSachRong(t *testing.T) {
	repo := &fakeArtworkRepo{}
	svc := &artworkService{artworkRepo: repo}

	err := svc.SetFeaturedBatch(context.Background(), nil, true)
	if err == nil {
		t.Fatal("muốn lỗi khi danh sách id rỗng, nhận nil")
	}
	if repo.batchCalls != 0 {
		t.Errorf("không nên gọi repository khi danh sách rỗng, đã gọi %d lần", repo.batchCalls)
	}
}

// Danh sách id hợp lệ phải được chuyển nguyên vẹn xuống repository, kèm đúng
// giá trị is_featured mong muốn.
func TestSetFeaturedBatchChuyenDungThamSo(t *testing.T) {
	repo := &fakeArtworkRepo{}
	svc := &artworkService{artworkRepo: repo}

	ids := []int64{3, 7, 9}
	if err := svc.SetFeaturedBatch(context.Background(), ids, true); err != nil {
		t.Fatalf("không muốn lỗi: %v", err)
	}

	if repo.batchCalls != 1 {
		t.Fatalf("muốn gọi repository đúng 1 lần, nhận %d", repo.batchCalls)
	}
	if len(repo.batchIDs) != len(ids) {
		t.Fatalf("muốn %d id, nhận %d", len(ids), len(repo.batchIDs))
	}
	for i, id := range ids {
		if repo.batchIDs[i] != id {
			t.Errorf("id[%d] = %d, muốn %d", i, repo.batchIDs[i], id)
		}
	}
	if !repo.batchValue {
		t.Error("muốn is_featured=true được truyền xuống repository")
	}
}

// Lỗi từ repository phải được trả nguyên lên trên, không bị nuốt.
func TestSetFeaturedBatchLoiRepositoryDuocTraLen(t *testing.T) {
	wantErr := errors.New("lỗi DB giả lập")
	repo := &fakeArtworkRepo{batchErr: wantErr}
	svc := &artworkService{artworkRepo: repo}

	err := svc.SetFeaturedBatch(context.Background(), []int64{1}, false)
	if !errors.Is(err, wantErr) {
		t.Fatalf("muốn lỗi %v, nhận %v", wantErr, err)
	}
}
