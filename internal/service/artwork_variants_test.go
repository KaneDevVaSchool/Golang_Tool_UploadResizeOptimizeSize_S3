package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"sort"
	"strings"
	"sync"
	"testing"

	"s3-upload-tool/internal/models"
)

// fakeUploadService ghi lại các lần UploadDerived thay vì gọi S3 thật.
type fakeUploadService struct {
	mu       sync.Mutex
	uploaded map[string]string // key -> contentType
	failOn   string            // key chứa chuỗi này sẽ lỗi
}

func newFakeUploadService() *fakeUploadService {
	return &fakeUploadService{uploaded: map[string]string{}}
}

func (f *fakeUploadService) UploadImage(context.Context, string, io.Reader, int64, int64) (*models.UploadResponse, error) {
	return nil, errors.New("không dùng trong test này")
}

func (f *fakeUploadService) UploadImageWithTransaction(context.Context, string, io.Reader, int64, int64) (*models.UploadResponse, *models.UploadRecord, error) {
	return nil, nil, errors.New("không dùng trong test này")
}

func (f *fakeUploadService) UploadDerived(_ context.Context, key string, data []byte, contentType string) (string, error) {
	if f.failOn != "" && strings.Contains(key, f.failOn) {
		return "", errors.New("lỗi upload giả lập")
	}
	if len(data) == 0 {
		return "", errors.New("dữ liệu rỗng")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.uploaded[key] = contentType
	return "https://cdn.test/" + key, nil
}

func (f *fakeUploadService) keys() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.uploaded))
	for k := range f.uploaded {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func testImageBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("tạo ảnh test lỗi: %v", err)
	}
	return buf.Bytes()
}

func TestBuildVariantsUploadsAllAndMapsURLs(t *testing.T) {
	fake := newFakeUploadService()
	svc := &artworkService{uploadService: fake}

	data := testImageBytes(t, 2000, 1500)
	file := BulkUploadFile{
		TempKey:  "tmp-1",
		FileName: "tranh.jpg",
		Reader:   bytes.NewReader(data),
		FileSize: int64(len(data)),
	}

	variants := svc.buildVariants(context.Background(), file, "vaschools-uploads/images/tranh-abc123.jpg")

	// 3 cỡ × 2 định dạng
	if len(variants) != 6 {
		t.Fatalf("có %d biến thể, muốn 6: %v", len(variants), variants)
	}

	for _, want := range []string{
		"thumb_webp", "thumb_jpg",
		"medium_webp", "medium_jpg",
		"large_webp", "large_jpg",
	} {
		url, ok := variants[want]
		if !ok {
			t.Errorf("thiếu khoá %s", want)
			continue
		}
		if !strings.HasPrefix(url, "https://cdn.test/") {
			t.Errorf("%s có URL lạ: %s", want, url)
		}
	}

	// Key biến thể phải bám theo key gốc, bỏ đuôi cũ và gắn hậu tố.
	keys := fake.keys()
	for _, k := range keys {
		if !strings.HasPrefix(k, "vaschools-uploads/images/tranh-abc123_") {
			t.Errorf("key %q không bám theo key gốc", k)
		}
		if strings.Contains(k, ".jpg_") {
			t.Errorf("key %q còn sót đuôi file gốc", k)
		}
	}
	t.Logf("đã upload: %v", keys)

	// Content-Type phải đúng, nếu sai trình duyệt sẽ tải về thay vì hiển thị.
	fake.mu.Lock()
	defer fake.mu.Unlock()
	for k, ct := range fake.uploaded {
		switch {
		case strings.HasSuffix(k, ".webp") && ct != "image/webp":
			t.Errorf("%s có Content-Type %q, muốn image/webp", k, ct)
		case strings.HasSuffix(k, ".jpg") && ct != "image/jpeg":
			t.Errorf("%s có Content-Type %q, muốn image/jpeg", k, ct)
		}
	}
}

// Ảnh nhỏ hơn mọi cỡ đích -> không sinh gì, dùng thẳng ảnh gốc.
func TestBuildVariantsTinyImageProducesNothing(t *testing.T) {
	fake := newFakeUploadService()
	svc := &artworkService{uploadService: fake}

	data := testImageBytes(t, 150, 120)
	file := BulkUploadFile{FileName: "nho.jpg", Reader: bytes.NewReader(data), FileSize: int64(len(data))}

	variants := svc.buildVariants(context.Background(), file, "vaschools-uploads/nho.jpg")

	if variants != nil {
		t.Errorf("muốn nil, nhận %v", variants)
	}
	if len(fake.keys()) != 0 {
		t.Errorf("không nên upload gì, đã upload: %v", fake.keys())
	}
}

// File không phải ảnh: không được panic, chỉ bỏ qua khâu biến thể.
func TestBuildVariantsInvalidImageDoesNotFail(t *testing.T) {
	fake := newFakeUploadService()
	svc := &artworkService{uploadService: fake}

	file := BulkUploadFile{
		FileName: "hong.jpg",
		Reader:   bytes.NewReader([]byte("không phải dữ liệu ảnh")),
	}

	variants := svc.buildVariants(context.Background(), file, "vaschools-uploads/hong.jpg")
	if variants != nil {
		t.Errorf("muốn nil khi ảnh hỏng, nhận %v", variants)
	}
}

// Một biến thể upload lỗi thì các biến thể còn lại vẫn phải dùng được -
// ảnh gốc đã an toàn trên S3, không có lý do bỏ luôn phần đã thành công.
func TestBuildVariantsPartialUploadFailure(t *testing.T) {
	fake := newFakeUploadService()
	fake.failOn = "_large." // key có dạng "<base>_large.webp" / "<base>_large.jpg"
	svc := &artworkService{uploadService: fake}

	data := testImageBytes(t, 2000, 1500)
	file := BulkUploadFile{FileName: "tranh.jpg", Reader: bytes.NewReader(data), FileSize: int64(len(data))}

	variants := svc.buildVariants(context.Background(), file, "vaschools-uploads/tranh.jpg")

	if len(variants) == 0 {
		t.Fatal("muốn giữ lại các biến thể thành công, nhận rỗng")
	}
	if _, ok := variants["thumb_webp"]; !ok {
		t.Error("thumb_webp phải còn dù large lỗi")
	}
	for key := range variants {
		if strings.HasPrefix(key, "large_") {
			t.Errorf("large không nên có mặt: %s", key)
		}
	}
}

// Reader đã bị đọc tới cuối (UploadImage chạy trước) vẫn phải sinh được biến
// thể - buildVariants có trách nhiệm tua lại từ đầu.
func TestBuildVariantsRewindsReader(t *testing.T) {
	fake := newFakeUploadService()
	svc := &artworkService{uploadService: fake}

	data := testImageBytes(t, 1200, 900)
	reader := bytes.NewReader(data)
	if _, err := io.Copy(io.Discard, reader); err != nil {
		t.Fatalf("đọc cạn reader lỗi: %v", err)
	}

	file := BulkUploadFile{FileName: "tranh.jpg", Reader: reader, FileSize: int64(len(data))}
	variants := svc.buildVariants(context.Background(), file, "vaschools-uploads/tranh.jpg")

	if len(variants) == 0 {
		t.Fatal("muốn sinh được biến thể sau khi tua lại, nhận rỗng")
	}
	fmt.Sprint(variants)
}
