package service

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"math"
	"runtime"
	"strings"
	"sync"

	// Encoder WebP. golang.org/x/image/webp chỉ giải mã được, không có
	// encoder; còn nativewebp thì chỉ hỗ trợ VP8L (lossless) - với tranh vẽ
	// và ảnh chụp, lossless cho ra file lớn gấp nhiều lần JPEG nên vô dụng ở
	// đây. Thư viện này encode lossy thật (VP8) qua WASM nhúng + purego, tức
	// vẫn không cần cgo, giữ nguyên được Dockerfile hiện tại.
	"github.com/gen2brain/webp"

	// Đăng ký decoder cho các định dạng ảnh đầu vào.
	_ "image/gif"
	_ "image/png"

	_ "golang.org/x/image/webp"
)

// VariantName định danh một cỡ ảnh dẫn xuất.
type VariantName string

const (
	VariantThumb  VariantName = "thumb"  // ô lưới gallery
	VariantMedium VariantName = "medium" // lightbox mở nhanh, màn hình vừa
	VariantLarge  VariantName = "large"  // màn hình lớn / retina
)

// variantSpec - cạnh dài tối đa của mỗi biến thể. Ảnh nhỏ hơn không bị phóng
// to (xem shouldSkip), nên tranh gốc nhỏ sẽ có ít biến thể hơn.
type variantSpec struct {
	name    VariantName
	maxEdge int
}

// variantSpecs xếp từ nhỏ tới lớn.
//
// Con số chọn theo cách ảnh thực sự được hiển thị:
//   - 400px: ô lưới rộng ~260-320px, nhân 1.5x cho màn hình retina.
//   - 1000px: đủ nét cho lightbox trên laptop mà vẫn mở gần như tức thì.
//   - 1600px: màn hình lớn và retina; trên mức này thì dùng thẳng ảnh gốc.
var variantSpecs = []variantSpec{
	{VariantThumb, 400},
	{VariantMedium, 1000},
	{VariantLarge, 1600},
}

// jpegQualityByVariant - thumb nhỏ nên chịu được chất lượng thấp hơn mà mắt
// khó nhận ra, đổi lại tiết kiệm đáng kể khi lưới có hàng chục ảnh.
var jpegQualityByVariant = map[VariantName]int{
	VariantThumb:  78,
	VariantMedium: 82,
	VariantLarge:  85,
}

const webpQuality = 80

// GeneratedVariant là một file ảnh dẫn xuất đã encode xong, sẵn sàng upload.
type GeneratedVariant struct {
	Name   VariantName
	Format string // "webp" | "jpg"
	Data   []byte
	Width  int
	Height int
}

// Key trả về hậu tố gắn vào s3 key gốc, ví dụ "thumb.webp".
func (v GeneratedVariant) Key() string {
	return fmt.Sprintf("%s.%s", v.Name, v.Format)
}

// GenerateVariants sinh toàn bộ biến thể từ bytes ảnh gốc.
//
// Với mỗi cỡ sinh hai định dạng: WebP (nhẹ hơn JPEG khoảng 25-35% ở cùng chất
// lượng) và JPEG làm dự phòng cho trình duyệt cũ - phía frontend dùng <picture>
// để trình duyệt tự chọn.
//
// Trả về danh sách rỗng (không phải lỗi) nếu ảnh gốc đã nhỏ hơn mọi cỡ đích:
// lúc đó dùng thẳng ảnh gốc là hợp lý nhất.
func GenerateVariants(ctx context.Context, original []byte) ([]GeneratedVariant, int, int, error) {
	src, _, err := image.Decode(bytes.NewReader(original))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("không giải mã được ảnh: %w", err)
	}

	b := src.Bounds()
	origW, origH := b.Dx(), b.Dy()
	if origW <= 0 || origH <= 0 {
		return nil, 0, 0, fmt.Errorf("ảnh có kích thước không hợp lệ: %dx%d", origW, origH)
	}

	// Chuyển một lần sang RGBA để mọi lần thu nhỏ sau đó đi theo đường nhanh
	// (đọc/ghi thẳng trên slice Pix) thay vì qua interface color.Color.
	rgba := toRGBA(src)

	var (
		mu      sync.Mutex
		results []GeneratedVariant
		wg      sync.WaitGroup
		firstEr error
	)

	for _, spec := range variantSpecs {
		if skipVariant(origW, origH, spec.maxEdge) {
			continue
		}

		w, h := fitWithin(origW, origH, spec.maxEdge)
		resized := resizeRGBA(ctx, rgba, w, h)

		// WebP và JPEG của cùng một cỡ encode song song - đây là phần tốn
		// CPU nhất và hai định dạng độc lập nhau.
		for _, format := range []string{"webp", "jpg"} {
			wg.Add(1)
			go func(spec variantSpec, format string, img *image.RGBA, w, h int) {
				defer wg.Done()

				data, err := encodeImage(img, format, jpegQualityByVariant[spec.name])
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					if firstEr == nil {
						firstEr = fmt.Errorf("encode %s/%s: %w", spec.name, format, err)
					}
					return
				}
				results = append(results, GeneratedVariant{
					Name:   spec.name,
					Format: format,
					Data:   data,
					Width:  w,
					Height: h,
				})
			}(spec, format, resized, w, h)
		}
	}

	wg.Wait()
	if firstEr != nil {
		return nil, origW, origH, firstEr
	}
	return results, origW, origH, nil
}

// skipVariant - không phóng to ảnh. Biến thể chỉ có nghĩa khi nó thực sự nhỏ
// hơn ảnh gốc; sinh bản "1600px" từ ảnh 800px chỉ tốn dung lượng mà không
// thêm chi tiết nào.
func skipVariant(origW, origH, maxEdge int) bool {
	longest := origW
	if origH > longest {
		longest = origH
	}
	return longest <= maxEdge
}

// fitWithin thu nhỏ giữ nguyên tỉ lệ sao cho cạnh dài bằng maxEdge.
func fitWithin(origW, origH, maxEdge int) (int, int) {
	if origW >= origH {
		h := int(math.Round(float64(origH) * float64(maxEdge) / float64(origW)))
		return maxEdge, max1(h)
	}
	w := int(math.Round(float64(origW) * float64(maxEdge) / float64(origH)))
	return max1(w), maxEdge
}

func max1(v int) int {
	if v < 1 {
		return 1
	}
	return v
}

// toRGBA đưa ảnh về *image.RGBA. Ảnh đã đúng kiểu và gốc tại (0,0) thì dùng lại.
func toRGBA(src image.Image) *image.RGBA {
	if r, ok := src.(*image.RGBA); ok && r.Bounds().Min == (image.Point{}) {
		return r
	}
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	return dst
}

// resizeRGBA thu nhỏ bằng nội suy bilinear, chia hàng cho nhiều goroutine.
//
// Khác với resizeBilinear trong image_resize.go (dùng img.At()/dst.Set() nên
// mỗi pixel phải đi qua interface color.Color và cấp phát kèm theo), bản này
// đọc ghi thẳng trên slice Pix. Cùng thuật toán, nhưng vì nay chạy cho mọi
// ảnh upload chứ không riêng luồng WordPress nên chi phí đó đáng để tránh.
func resizeRGBA(ctx context.Context, src *image.RGBA, width, height int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	if width <= 0 || height <= 0 || sw <= 0 || sh <= 0 {
		return dst
	}

	// -1 để pixel đích cuối cùng ánh xạ đúng vào pixel nguồn cuối cùng.
	xRatio := float64(sw-1) / float64(width)
	yRatio := float64(sh-1) / float64(height)

	workers := runtime.NumCPU()
	if workers > height {
		workers = height
	}
	if workers < 1 {
		workers = 1
	}

	var wg sync.WaitGroup
	rows := make(chan int, height)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for y := range rows {
				// Kiểm tra huỷ theo từng nhóm hàng, không phải mỗi pixel.
				if y%64 == 0 {
					select {
					case <-ctx.Done():
						return
					default:
					}
				}
				resizeRow(src, dst, y, width, sw, sh, xRatio, yRatio)
			}
		}()
	}

	go func() {
		defer close(rows)
		for y := 0; y < height; y++ {
			select {
			case <-ctx.Done():
				return
			case rows <- y:
			}
		}
	}()

	wg.Wait()
	return dst
}

// resizeRow nội suy một hàng của ảnh đích.
func resizeRow(src, dst *image.RGBA, y, width, sw, sh int, xRatio, yRatio float64) {
	srcY := yRatio * float64(y)
	y1 := int(srcY)
	y2 := y1 + 1
	if y2 >= sh {
		y2 = sh - 1
	}
	fy := srcY - float64(y1)

	row1 := y1 * src.Stride
	row2 := y2 * src.Stride
	dstRow := y * dst.Stride

	for x := 0; x < width; x++ {
		srcX := xRatio * float64(x)
		x1 := int(srcX)
		x2 := x1 + 1
		if x2 >= sw {
			x2 = sw - 1
		}
		fx := srcX - float64(x1)

		i11 := row1 + x1*4
		i21 := row1 + x2*4
		i12 := row2 + x1*4
		i22 := row2 + x2*4
		o := dstRow + x*4

		// 4 kênh R,G,B,A - nội suy ngang trước rồi dọc, giống
		// bilinearInterpolate nhưng trên byte thay vì color.Color.
		for c := 0; c < 4; c++ {
			top := float64(src.Pix[i11+c])*(1-fx) + float64(src.Pix[i21+c])*fx
			bot := float64(src.Pix[i12+c])*(1-fx) + float64(src.Pix[i22+c])*fx
			dst.Pix[o+c] = uint8(math.Round(top*(1-fy) + bot*fy))
		}
	}
}

// encodeImage encode ảnh ra định dạng đích.
func encodeImage(img *image.RGBA, format string, jpegQuality int) ([]byte, error) {
	var buf bytes.Buffer

	switch strings.ToLower(format) {
	case "webp":
		// Method 4 là mặc định của libwebp: cân bằng giữa thời gian encode và
		// dung lượng. Mức cao hơn chỉ tiết kiệm thêm vài phần trăm nhưng chậm
		// hơn hẳn, không đáng khi phải xử lý cả loạt ảnh lúc upload.
		if err := webp.Encode(&buf, img, webp.Options{Quality: webpQuality, Method: 4}); err != nil {
			return nil, err
		}
	case "jpg", "jpeg":
		// JPEG không có alpha: ghép nền trắng trước, nếu không vùng trong
		// suốt của PNG sẽ ra đen.
		if err := jpeg.Encode(&buf, flattenAlpha(img), &jpeg.Options{Quality: jpegQuality}); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("định dạng không hỗ trợ: %s", format)
	}

	return buf.Bytes(), nil
}

// flattenAlpha ghép ảnh lên nền trắng khi ảnh có vùng trong suốt.
func flattenAlpha(img *image.RGBA) image.Image {
	opaque := true
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] != 0xFF {
			opaque = false
			break
		}
	}
	if opaque {
		return img
	}

	out := image.NewRGBA(img.Bounds())
	draw.Draw(out, out.Bounds(), image.NewUniform(image.White.C), image.Point{}, draw.Src)
	draw.Draw(out, out.Bounds(), img, img.Bounds().Min, draw.Over)
	return out
}
