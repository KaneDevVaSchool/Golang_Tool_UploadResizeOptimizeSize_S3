package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"s3-upload-tool/internal/models"
)

// Dùng lại fakeArtworkRepo khai báo ở artwork_bulk_featured_test.go (cùng
// package service) - field listPages/listErr/listFilters riêng cho List,
// không đụng tới batchIDs/batchErr mà các test bulk-featured đang dùng.

func newSitemapArtwork(id int64, updatedAt time.Time) *models.Artwork {
	return &models.Artwork{ID: id, UpdatedAt: updatedAt}
}

// Danh sách ít hơn 1 trang: chỉ cần gọi List đúng 1 lần là đủ, không lặp thêm
// một lần rỗng vô ích.
func TestListPublishedForSitemapItHon1Trang(t *testing.T) {
	now := time.Now()
	repo := &fakeArtworkRepo{
		listPages: [][]*models.Artwork{
			{newSitemapArtwork(1, now), newSitemapArtwork(2, now), newSitemapArtwork(3, now), newSitemapArtwork(4, now), newSitemapArtwork(5, now)},
		},
	}
	svc := &artworkService{artworkRepo: repo}

	result, err := svc.ListPublishedForSitemap(context.Background())
	if err != nil {
		t.Fatalf("không muốn lỗi: %v", err)
	}
	if len(result) != 5 {
		t.Fatalf("muốn 5 tác phẩm, nhận %d", len(result))
	}
	if len(repo.listFilters) != 1 {
		t.Fatalf("muốn gọi List đúng 1 lần, nhận %d", len(repo.listFilters))
	}
}

// Danh sách vượt quá 1 trang (đủ 100 item ở trang đầu, ép hàm phải lặp sang
// trang 2) phải gom đủ toàn bộ và gọi List đúng số lần tương ứng.
func TestListPublishedForSitemapNhieuTrang(t *testing.T) {
	now := time.Now()
	page1 := make([]*models.Artwork, 100)
	for i := range page1 {
		page1[i] = newSitemapArtwork(int64(i+1), now)
	}
	page2 := make([]*models.Artwork, 50)
	for i := range page2 {
		page2[i] = newSitemapArtwork(int64(100+i+1), now)
	}
	repo := &fakeArtworkRepo{listPages: [][]*models.Artwork{page1, page2}}
	svc := &artworkService{artworkRepo: repo}

	result, err := svc.ListPublishedForSitemap(context.Background())
	if err != nil {
		t.Fatalf("không muốn lỗi: %v", err)
	}
	if len(result) != 150 {
		t.Fatalf("muốn 150 tác phẩm, nhận %d", len(result))
	}
	if len(repo.listFilters) != 2 {
		t.Fatalf("muốn gọi List đúng 2 lần, nhận %d", len(repo.listFilters))
	}
}

// Lỗi từ repository phải được bọc ngữ cảnh và trả nguyên lên trên, không
// panic, không bị nuốt.
func TestListPublishedForSitemapLoiRepository(t *testing.T) {
	wantErr := errors.New("lỗi DB giả lập")
	repo := &fakeArtworkRepo{listErr: wantErr}
	svc := &artworkService{artworkRepo: repo}

	_, err := svc.ListPublishedForSitemap(context.Background())
	if err == nil {
		t.Fatal("muốn lỗi, nhận nil")
	}
	if !errors.Is(err, wantErr) {
		t.Fatalf("muốn lỗi bọc %v, nhận %v", wantErr, err)
	}
}

// Chưa có tác phẩm nào publish: trả slice rỗng, không lỗi.
func TestListPublishedForSitemapDanhSachRong(t *testing.T) {
	repo := &fakeArtworkRepo{listPages: [][]*models.Artwork{{}}}
	svc := &artworkService{artworkRepo: repo}

	result, err := svc.ListPublishedForSitemap(context.Background())
	if err != nil {
		t.Fatalf("không muốn lỗi: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("muốn danh sách rỗng, nhận %d item", len(result))
	}
}

// Filter gửi xuống repository phải luôn ép IsPublished=true - sitemap không
// bao giờ được liệt kê tác phẩm chưa publish.
func TestListPublishedForSitemapEpIsPublished(t *testing.T) {
	repo := &fakeArtworkRepo{listPages: [][]*models.Artwork{{}}}
	svc := &artworkService{artworkRepo: repo}

	if _, err := svc.ListPublishedForSitemap(context.Background()); err != nil {
		t.Fatalf("không muốn lỗi: %v", err)
	}
	if len(repo.listFilters) != 1 {
		t.Fatalf("muốn gọi List đúng 1 lần, nhận %d", len(repo.listFilters))
	}
	filter := repo.listFilters[0]
	if filter.IsPublished == nil || !*filter.IsPublished {
		t.Error("muốn filter.IsPublished = true, nhận khác")
	}
}
