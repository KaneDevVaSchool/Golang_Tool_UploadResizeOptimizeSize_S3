package service

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

// makeTestJPEG tạo ảnh gradient để nén ra kích thước gần với ảnh thật (ảnh
// một màu nén quá tốt, không phản ánh đúng tỉ lệ giữa các định dạng).
func makeTestJPEG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 255) / w),
				G: uint8((y * 255) / h),
				B: uint8(((x + y) * 255) / (w + h)),
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("tạo ảnh test lỗi: %v", err)
	}
	return buf.Bytes()
}

func TestGenerateVariantsProducesAllSizesAndFormats(t *testing.T) {
	original := makeTestJPEG(t, 2400, 1800)

	variants, w, h, err := GenerateVariants(context.Background(), original)
	if err != nil {
		t.Fatalf("GenerateVariants lỗi: %v", err)
	}
	if w != 2400 || h != 1800 {
		t.Errorf("kích thước gốc = %dx%d, muốn 2400x1800", w, h)
	}

	// 3 cỡ × 2 định dạng
	if len(variants) != 6 {
		t.Fatalf("có %d biến thể, muốn 6", len(variants))
	}

	seen := map[string]GeneratedVariant{}
	for _, v := range variants {
		seen[v.Key()] = v
	}
	for _, key := range []string{
		"thumb.webp", "thumb.jpg",
		"medium.webp", "medium.jpg",
		"large.webp", "large.jpg",
	} {
		v, ok := seen[key]
		if !ok {
			t.Errorf("thiếu biến thể %s", key)
			continue
		}
		if len(v.Data) == 0 {
			t.Errorf("%s rỗng", key)
		}
	}

	// Ảnh ngang 2400x1800 -> cạnh dài đúng bằng maxEdge, cạnh ngắn giữ tỉ lệ 3:4.
	if got := seen["thumb.webp"]; got.Width != 400 || got.Height != 300 {
		t.Errorf("thumb = %dx%d, muốn 400x300", got.Width, got.Height)
	}
	if got := seen["medium.webp"]; got.Width != 1000 || got.Height != 750 {
		t.Errorf("medium = %dx%d, muốn 1000x750", got.Width, got.Height)
	}
	if got := seen["large.webp"]; got.Width != 1600 || got.Height != 1200 {
		t.Errorf("large = %dx%d, muốn 1600x1200", got.Width, got.Height)
	}

	// Lý do chính để sinh WebP: phải nhỏ hơn JPEG cùng cỡ.
	for _, name := range []string{"thumb", "medium", "large"} {
		webp := len(seen[name+".webp"].Data)
		jpg := len(seen[name+".jpg"].Data)
		t.Logf("%-7s webp=%6dB  jpg=%6dB  (webp nhỏ hơn %d%%)", name, webp, jpg, 100-webp*100/jpg)
		if webp >= jpg {
			t.Errorf("%s: webp (%dB) không nhỏ hơn jpg (%dB)", name, webp, jpg)
		}
	}

	// Thumb phải nhỏ hơn ảnh gốc rất nhiều - đây mới là điều khiến lưới nhanh lên.
	if thumb := len(seen["thumb.webp"].Data); thumb > len(original)/5 {
		t.Errorf("thumb.webp = %dB, quá lớn so với gốc %dB", thumb, len(original))
	}
}

func TestGenerateVariantsPortraitOrientation(t *testing.T) {
	original := makeTestJPEG(t, 1200, 1600) // ảnh dọc

	variants, _, _, err := GenerateVariants(context.Background(), original)
	if err != nil {
		t.Fatalf("GenerateVariants lỗi: %v", err)
	}

	for _, v := range variants {
		if v.Name != VariantThumb || v.Format != "webp" {
			continue
		}
		// Ảnh dọc: cạnh dài là chiều cao.
		if v.Height != 400 {
			t.Errorf("thumb dọc height = %d, muốn 400", v.Height)
		}
		if v.Width != 300 {
			t.Errorf("thumb dọc width = %d, muốn 300", v.Width)
		}
	}
}

// Không phóng to ảnh nhỏ: sinh bản "large" từ ảnh 500px chỉ tốn chỗ.
func TestGenerateVariantsSkipsUpscaling(t *testing.T) {
	original := makeTestJPEG(t, 500, 375)

	variants, _, _, err := GenerateVariants(context.Background(), original)
	if err != nil {
		t.Fatalf("GenerateVariants lỗi: %v", err)
	}

	for _, v := range variants {
		if v.Name != VariantThumb {
			t.Errorf("ảnh 500px không nên sinh biến thể %q", v.Name)
		}
		if v.Width > 500 || v.Height > 375 {
			t.Errorf("%s bị phóng to thành %dx%d", v.Key(), v.Width, v.Height)
		}
	}
	// 500 > 400 nên vẫn có thumb; 500 <= 1000 và <= 1600 nên không có medium/large.
	if len(variants) != 2 {
		t.Errorf("có %d biến thể, muốn 2 (thumb webp+jpg)", len(variants))
	}
}

// Ảnh nhỏ hơn mọi cỡ đích -> không có biến thể nào, dùng thẳng ảnh gốc.
func TestGenerateVariantsTinyImage(t *testing.T) {
	original := makeTestJPEG(t, 200, 150)

	variants, w, h, err := GenerateVariants(context.Background(), original)
	if err != nil {
		t.Fatalf("GenerateVariants lỗi: %v", err)
	}
	if len(variants) != 0 {
		t.Errorf("có %d biến thể, muốn 0", len(variants))
	}
	if w != 200 || h != 150 {
		t.Errorf("kích thước = %dx%d, muốn 200x150", w, h)
	}
}

// PNG trong suốt chuyển sang JPEG phải ghép nền trắng, không ra đen.
func TestGenerateVariantsFlattensTransparency(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 900, 900))
	// Toàn bộ trong suốt hoàn toàn.
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("tạo png lỗi: %v", err)
	}

	variants, _, _, err := GenerateVariants(context.Background(), buf.Bytes())
	if err != nil {
		t.Fatalf("GenerateVariants lỗi: %v", err)
	}

	for _, v := range variants {
		if v.Format != "jpg" {
			continue
		}
		decoded, err := jpeg.Decode(bytes.NewReader(v.Data))
		if err != nil {
			t.Fatalf("giải mã %s lỗi: %v", v.Key(), err)
		}
		r, g, b, _ := decoded.At(v.Width/2, v.Height/2).RGBA()
		// Nền trắng, cho phép sai số do nén JPEG.
		if r>>8 < 240 || g>>8 < 240 || b>>8 < 240 {
			t.Errorf("%s: pixel giữa = (%d,%d,%d), muốn gần trắng", v.Key(), r>>8, g>>8, b>>8)
		}
	}
}

func TestGenerateVariantsRejectsInvalidData(t *testing.T) {
	_, _, _, err := GenerateVariants(context.Background(), []byte("đây không phải ảnh"))
	if err == nil {
		t.Fatal("muốn lỗi với dữ liệu không phải ảnh, nhưng err == nil")
	}
}

// Huỷ context giữa chừng không được làm treo hay panic.
func TestGenerateVariantsRespectsCancelledContext(t *testing.T) {
	original := makeTestJPEG(t, 2000, 1500)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Chỉ cần không treo và không panic; kết quả có thể dở dang.
	_, _, _, _ = GenerateVariants(ctx, original)
}
