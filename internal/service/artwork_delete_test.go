package service

import (
	"sort"
	"testing"

	"s3-upload-tool/internal/models"
	"s3-upload-tool/internal/utils"
)

func TestCollectArtworkS3Keys(t *testing.T) {
	t.Parallel()

	bucket := "my-bucket"
	keyFromURL := func(u string) string { return utils.ParseS3ObjectKey(u, bucket) }

	thumb := "https://my-bucket.s3.ap-southeast-1.amazonaws.com/prefix/a_thumb.jpg"
	artwork := &models.Artwork{
		S3Key: "prefix/a.jpg",
		Variants: models.ArtworkVariants{
			"thumb_webp": "https://my-bucket.s3.ap-southeast-1.amazonaws.com/prefix/a_thumb.webp",
		},
		ThumbnailURL: &thumb,
	}

	got := collectArtworkS3Keys(artwork, keyFromURL)
	sort.Strings(got)
	want := []string{"prefix/a.jpg", "prefix/a_thumb.jpg", "prefix/a_thumb.webp"}
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}
