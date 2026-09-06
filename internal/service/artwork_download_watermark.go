package service

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gen2brain/webp"
	xdraw "golang.org/x/image/draw"

	_ "image/gif"
	_ "golang.org/x/image/webp"
)

var (
	watermarkOnce sync.Once
	watermarkSrc  image.Image
	watermarkLoad error
)

func loadVASWatermarkMark() (image.Image, error) {
	watermarkOnce.Do(func() {
		candidates := []string{
			filepath.Join("web", "public", "images", "vas-white-mark.png"),
			filepath.Join("web", "dist", "images", "vas-white-mark.png"),
		}
		for _, path := range candidates {
			raw, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			img, _, err := image.Decode(bytes.NewReader(raw))
			if err != nil {
				watermarkLoad = fmt.Errorf("decode watermark %s: %w", path, err)
				return
			}
			watermarkSrc = img
			return
		}
		watermarkLoad = fmt.Errorf("không tìm thấy vas-white-mark.png trong web/public hoặc web/dist")
	})
	return watermarkSrc, watermarkLoad
}

// ApplyArtworkDownloadWatermark ghép logo VAS nhỏ sát góc dưới-phải trước khi
// trả file tải về. ext là đuôi file gốc (vd ".jpg") để giữ định dạng đầu ra.
func ApplyArtworkDownloadWatermark(original []byte, ext string) ([]byte, string, error) {
	src, _, err := image.Decode(bytes.NewReader(original))
	if err != nil {
		return nil, "", err
	}

	mark, err := loadVASWatermarkMark()
	if err != nil {
		return nil, "", err
	}

	bounds := src.Bounds()
	canvas := image.NewRGBA(bounds)
	draw.Draw(canvas, bounds, src, bounds.Min, draw.Src)

	minDim := bounds.Dx()
	if bounds.Dy() < minDim {
		minDim = bounds.Dy()
	}
	padding := int(math.Max(12, float64(minDim)*0.022))
	targetW := int(math.Max(40, float64(minDim)*0.11))
	scaled := scaleWatermarkMark(mark, targetW)
	if scaled == nil {
		return nil, "", fmt.Errorf("không scale được watermark")
	}

	wmBounds := scaled.Bounds()
	x := bounds.Max.X - wmBounds.Dx() - padding
	y := bounds.Max.Y - wmBounds.Dy() - padding
	if x < bounds.Min.X {
		x = bounds.Min.X + padding
	}
	if y < bounds.Min.Y {
		y = bounds.Min.Y + padding
	}

	// Bóng mờ nhẹ giúp logo đọc được trên nền sáng lẫn tối.
	shadow := tintWatermarkLayer(scaled, color.NRGBA{R: 0, G: 0, B: 0, A: 90})
	draw.Draw(canvas, wmBounds.Add(image.Pt(x+2, y+2)), shadow, wmBounds.Min, draw.Over)
	draw.Draw(canvas, wmBounds.Add(image.Pt(x, y)), scaled, wmBounds.Min, draw.Over)

	format := strings.TrimPrefix(strings.ToLower(ext), ".")
	if format == "" {
		format = "jpg"
	}

	out, contentType, err := encodeWatermarkedImage(canvas, format)
	if err != nil {
		return nil, "", err
	}
	return out, contentType, nil
}

func scaleWatermarkMark(src image.Image, targetWidth int) *image.NRGBA {
	srcB := src.Bounds()
	if srcB.Dx() <= 0 || targetWidth <= 0 {
		return nil
	}
	ratio := float64(targetWidth) / float64(srcB.Dx())
	targetHeight := int(math.Max(1, math.Round(float64(srcB.Dy())*ratio)))
	layer := logoMarkLayer(src)
	dst := image.NewNRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), layer, layer.Bounds(), draw.Over, nil)
	return dst
}

// logoMarkLayer: nền đen trong file PNG -> trong suốt; phần trắng giữ ~78% opacity.
func logoMarkLayer(src image.Image) *image.NRGBA {
	b := src.Bounds()
	out := image.NewNRGBA(b)
	const markAlpha = 200
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r8, g8, b8, _ := src.At(x, y).RGBA()
			sum := int(r8>>8) + int(g8>>8) + int(b8>>8)
			if sum < 48 {
				continue
			}
			out.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: markAlpha})
		}
	}
	return out
}

func tintWatermarkLayer(src *image.NRGBA, tint color.NRGBA) *image.NRGBA {
	b := src.Bounds()
	out := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := src.At(x, y).RGBA()
			if a == 0 {
				continue
			}
			alpha := uint8(a >> 8)
			out.SetNRGBA(x, y, color.NRGBA{R: tint.R, G: tint.G, B: tint.B, A: uint8(int(alpha) * int(tint.A) / 255)})
		}
	}
	return out
}

func encodeWatermarkedImage(img *image.RGBA, format string) ([]byte, string, error) {
	var buf bytes.Buffer
	switch format {
	case "png":
		if err := png.Encode(&buf, img); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "image/png", nil
	case "webp":
		if err := webp.Encode(&buf, img, webp.Options{Quality: 88, Method: 4}); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "image/webp", nil
	case "jpg", "jpeg":
		if err := jpeg.Encode(&buf, flattenAlpha(img), &jpeg.Options{Quality: 92}); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "image/jpeg", nil
	default:
		if err := jpeg.Encode(&buf, flattenAlpha(img), &jpeg.Options{Quality: 92}); err != nil {
			return nil, "", err
		}
		return buf.Bytes(), "image/jpeg", nil
	}
}
