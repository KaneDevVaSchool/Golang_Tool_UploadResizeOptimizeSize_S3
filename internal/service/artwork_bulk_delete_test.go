package service

import (
	"context"
	"errors"
	"testing"

	"s3-upload-tool/internal/models"
)

// fakeDeleteArtworkRepo hỗ trợ đúng 2 phương thức DeleteArtwork cần:
// GetByID (đọc artwork trước khi xoá S3) và Delete (xoá bản ghi). Các
// phương thức khác của ArtworkRepository không dùng trong test này.
type fakeDeleteArtworkRepo struct {
	fakeArtworkRepo
	artworks   map[int64]*models.Artwork
	deletedIDs []int64
}

func (f *fakeDeleteArtworkRepo) GetByID(_ context.Context, id int64) (*models.Artwork, error) {
	a, ok := f.artworks[id]
	if !ok {
		return nil, nil
	}
	return a, nil
}

func (f *fakeDeleteArtworkRepo) Delete(_ context.Context, id int64) error {
	f.deletedIDs = append(f.deletedIDs, id)
	delete(f.artworks, id)
	return nil
}

// fakeDeleteUploadService giả lập DeleteObject - key chứa failOn sẽ lỗi, mô
// phỏng 1 tác phẩm trong lô gặp sự cố S3 (mất mạng, key không tồn tại...).
type fakeDeleteUploadService struct {
	fakeUploadService
	failOn string
}

func (f *fakeDeleteUploadService) DeleteObject(_ context.Context, key string) error {
	if f.failOn != "" && key == f.failOn {
		return errors.New("lỗi xoá S3 giả lập")
	}
	return nil
}

// DeleteArtworkBatch: một tác phẩm lỗi xoá S3 không được chặn các tác phẩm
// còn lại trong lô - đúng nguyên tắc "lỗi phụ trợ không hỏng thao tác chính"
// (ở đây thao tác chính là xoá bản ghi DB của TỪNG tác phẩm, độc lập nhau).
func TestDeleteArtworkBatchMotItemLoiKhongChanCaLo(t *testing.T) {
	repo := &fakeDeleteArtworkRepo{
		artworks: map[int64]*models.Artwork{
			1: {ID: 1, S3Key: "ok-1.jpg"},
			2: {ID: 2, S3Key: "loi.jpg"},
			3: {ID: 3, S3Key: "ok-3.jpg"},
		},
	}
	uploadSvc := &fakeDeleteUploadService{failOn: "loi.jpg"}
	svc := &artworkService{artworkRepo: repo, uploadService: uploadSvc}

	deleted, err := svc.DeleteArtworkBatch(context.Background(), []int64{1, 2, 3})
	if err != nil {
		t.Fatalf("không muốn lỗi ở tầng batch: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("muốn xoá được 2/3 tác phẩm, nhận %d", deleted)
	}
	if _, stillThere := repo.artworks[2]; !stillThere {
		t.Error("tác phẩm lỗi xoá S3 không được xoá khỏi DB, nhưng đã bị xoá")
	}
	if len(repo.artworks) != 1 {
		t.Errorf("muốn còn lại đúng 1 tác phẩm chưa xoá, nhận %d", len(repo.artworks))
	}
}

// Danh sách rỗng phải bị từ chối ở tầng service, không chạm repository.
func TestDeleteArtworkBatchTuChoiDanhSachRong(t *testing.T) {
	repo := &fakeDeleteArtworkRepo{artworks: map[int64]*models.Artwork{}}
	svc := &artworkService{artworkRepo: repo, uploadService: &fakeDeleteUploadService{}}

	if _, err := svc.DeleteArtworkBatch(context.Background(), nil); err == nil {
		t.Fatal("muốn lỗi khi danh sách id rỗng, nhận nil")
	}
}
