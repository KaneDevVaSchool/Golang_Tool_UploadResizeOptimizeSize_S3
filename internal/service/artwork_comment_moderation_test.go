package service

import (
	"context"
	"errors"
	"testing"

	"s3-upload-tool/internal/models"
)

// fakeCommentRepoForModeration ghi lại tham số của lần gọi SetHidden/ListByArtwork
// gần nhất - đủ để test SetCommentHidden/ListComments không cần MySQL thật.
// Implement đầy đủ repository.CommentRepository (chỉ 6 method, không cần embed
// interface rỗng) - method không dùng tới trong file này chỉ trả no-op.
type fakeCommentRepoForModeration struct {
	setHiddenID        int64
	setHiddenArtworkID int64
	setHiddenValue     bool
	setHiddenErr       error

	listArtworkID     int64
	listIncludeHidden bool
	listResult        []*models.ArtworkComment
	listErr           error
}

func (f *fakeCommentRepoForModeration) Create(ctx context.Context, comment *models.ArtworkComment) (*models.ArtworkComment, error) {
	return comment, nil
}

func (f *fakeCommentRepoForModeration) ListByArtwork(ctx context.Context, artworkID int64, includeHidden bool) ([]*models.ArtworkComment, error) {
	f.listArtworkID = artworkID
	f.listIncludeHidden = includeHidden
	return f.listResult, f.listErr
}

func (f *fakeCommentRepoForModeration) DeleteOwned(ctx context.Context, id, artworkID int64, visitorToken string) (bool, error) {
	return false, nil
}

func (f *fakeCommentRepoForModeration) SetHidden(ctx context.Context, id, artworkID int64, hidden bool) error {
	f.setHiddenID = id
	f.setHiddenArtworkID = artworkID
	f.setHiddenValue = hidden
	return f.setHiddenErr
}

func (f *fakeCommentRepoForModeration) CountByArtwork(ctx context.Context, artworkID int64) (int64, error) {
	return 0, nil
}

func (f *fakeCommentRepoForModeration) CountByArtworkBatch(ctx context.Context, artworkIDs []int64) (map[int64]int64, error) {
	return nil, nil
}

func TestArtworkService_SetCommentHidden_ScopesToArtworkID(t *testing.T) {
	t.Parallel()

	fake := &fakeCommentRepoForModeration{}
	svc := &artworkService{commentRepo: fake}

	if err := svc.SetCommentHidden(context.Background(), 42, 7, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Quan trọng nhất: commentID và artworkID phải truyền ĐÚNG THỨ TỰ xuống
	// repo, vì SetHidden dùng cả hai trong WHERE để chặn admin ẩn nhầm bình
	// luận của tác phẩm khác qua URL /artworks/{id}/comments/{commentID}.
	if fake.setHiddenID != 7 {
		t.Errorf("setHiddenID = %d, want 7 (commentID)", fake.setHiddenID)
	}
	if fake.setHiddenArtworkID != 42 {
		t.Errorf("setHiddenArtworkID = %d, want 42 (artworkID)", fake.setHiddenArtworkID)
	}
	if !fake.setHiddenValue {
		t.Errorf("setHiddenValue = false, want true")
	}
}

func TestArtworkService_SetCommentHidden_PropagatesRepoError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("comment not found: id=7, artwork_id=42")
	fake := &fakeCommentRepoForModeration{setHiddenErr: wantErr}
	svc := &artworkService{commentRepo: fake}

	err := svc.SetCommentHidden(context.Background(), 42, 7, false)
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
}

func TestArtworkService_ListComments_IncludesHidden(t *testing.T) {
	t.Parallel()

	fake := &fakeCommentRepoForModeration{
		listResult: []*models.ArtworkComment{{ID: 1, IsHidden: true}, {ID: 2, IsHidden: false}},
	}
	svc := &artworkService{commentRepo: fake}

	got, err := svc.ListComments(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Màn hình kiểm duyệt PHẢI thấy cả bình luận đã ẩn (để bấm hiện lại) -
	// khác PublicHandler.HandleListComments luôn lọc includeHidden=false.
	if !fake.listIncludeHidden {
		t.Errorf("listIncludeHidden = false, want true - moderation phải thấy cả comment đã ẩn")
	}
	if fake.listArtworkID != 42 {
		t.Errorf("listArtworkID = %d, want 42", fake.listArtworkID)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
}
