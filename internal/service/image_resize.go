package service

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"s3-upload-tool/internal/metrics"
	"s3-upload-tool/internal/utils"

	_ "golang.org/x/image/webp" // Support WebP format
)

// ImageSize represents a resized image size
type ImageSize struct {
	Name   string
	Width  int
	Height int
	Path   string
	URL    string
}

// ImageResizeService handles image resizing operations
type ImageResizeService struct {
	wpUploadsDir string
	baseURL      string
	sizes        []ImageSizeConfig
	optimizer    *ImageOptimizer
}

// ImageSizeConfig defines image size configuration
type ImageSizeConfig struct {
	Name   string
	Width  int
	Height int
}

// NewImageResizeService creates a new image resize service
func NewImageResizeService(wpUploadsDir, baseURL string, sizes []ImageSizeConfig, optimizer *ImageOptimizer) *ImageResizeService {
	return &ImageResizeService{
		wpUploadsDir: wpUploadsDir,
		baseURL:      baseURL,
		sizes:        sizes,
		optimizer:    optimizer,
	}
}

// ResizeImage resizes an image and creates multiple sizes
func (s *ImageResizeService) ResizeImage(ctx context.Context, originalPath, filename string) ([]ImageSize, error) {
	// Check cancellation before starting
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("resize cancelled: %w", ctx.Err())
	default:
	}

	// Read original image
	file, err := os.Open(originalPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open original image %s: %w", originalPath, err)
	}
	defer file.Close()

	// Reset file pointer
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek file: %w", err)
	}

	// Check cancellation after file operations
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("resize cancelled: %w", ctx.Err())
	default:
	}

	img, format, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image %s: %w", originalPath, err)
	}

	originalBounds := img.Bounds()
	originalWidth := originalBounds.Dx()
	originalHeight := originalBounds.Dy()

	log.Printf("[ImageResize] Original image: %dx%d, format: %s", originalWidth, originalHeight, format)

	// Create year/month directory structure (WordPress style)
	year := utils.GetCurrentYear()
	month := utils.GetCurrentMonth()
	uploadPath := filepath.Join(s.wpUploadsDir, year, month)
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory %s: %w", uploadPath, err)
	}

	baseName := strings.TrimSuffix(filename, filepath.Ext(filename))
	ext := filepath.Ext(filename)

	var results []ImageSize

	// Process each size
	for i, sizeConfig := range s.sizes {
		// Check context cancellation periodically (every 5 sizes)
		if i%5 == 0 {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("resize cancelled: %w", ctx.Err())
			default:
			}
		}
		// Check cancellation periodically
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("resize cancelled during processing: %w", ctx.Err())
		default:
		}

		resizedImg := s.resizeImage(ctx, img, sizeConfig.Width, sizeConfig.Height, originalWidth, originalHeight)

		// Generate filename for this size
		sizeFilename := s.generateSizeFilename(baseName, ext, sizeConfig.Name)
		sizePath := filepath.Join(uploadPath, sizeFilename)

		// Save resized image (temporary path for optimization) with error recovery
		tempPath := sizePath + ".tmp"
		maxSaveRetries := 2
		var saveErr error
		for attempt := 1; attempt <= maxSaveRetries; attempt++ {
			saveErr = s.saveImage(resizedImg, tempPath, format)
			if saveErr == nil {
				if attempt > 1 {
					log.Printf("[ImageResize] Successfully saved %s after %d attempts", sizeConfig.Name, attempt)
				}
				break
			}
			log.Printf("[ImageResize] Error saving %s (attempt %d/%d): %v", sizeConfig.Name, attempt, maxSaveRetries, saveErr)
		}
		if saveErr != nil {
			log.Printf("[ImageResize] Failed to save %s after %d attempts, skipping this size", sizeConfig.Name, maxSaveRetries)
			continue
		}

		// Optimize image if optimizer is available
		if s.optimizer != nil {
			optResult, err := s.optimizer.OptimizeImage(ctx, tempPath, sizePath)
			if err != nil {
				log.Printf("[ImageResize] Optimization failed for %s (original: %s, size: %dx%d), using original: %v",
					sizeConfig.Name, originalPath, sizeConfig.Width, sizeConfig.Height, err)
				metrics.GetMetrics().RecordError("image_optimization")

				// Fallback: move temp file to final location
				if renameErr := os.Rename(tempPath, sizePath); renameErr != nil {
					log.Printf("[ImageResize] Failed to rename temp file %s to %s: %v", tempPath, sizePath, renameErr)
					// Try to cleanup temp file if rename failed
					if removeErr := os.Remove(tempPath); removeErr != nil {
						log.Printf("[ImageResize] Failed to remove temp file after rename failure: %v", removeErr)
					}
				} else {
					log.Printf("[ImageResize] Fallback: using unoptimized version for %s", sizeConfig.Name)
				}
			} else {
				// Remove temp file
				if removeErr := os.Remove(tempPath); removeErr != nil {
					log.Printf("[ImageResize] Failed to remove temp file %s: %v", tempPath, removeErr)
				}
				log.Printf("[ImageResize] Optimized %s: saved %.2f%% (%d bytes)",
					sizeConfig.Name, optResult.SavedPercent, optResult.SavedBytes)

				// Record metrics
				metrics.GetMetrics().RecordImageOptimize(optResult.SavedBytes)
			}
		} else {
			// No optimizer, just rename temp to final
			if err := os.Rename(tempPath, sizePath); err != nil {
				log.Printf("[ImageResize] Failed to rename temp file %s to %s: %v", tempPath, sizePath, err)
				// Try to cleanup temp file
				if removeErr := os.Remove(tempPath); removeErr != nil {
					log.Printf("[ImageResize] Failed to remove temp file after rename failure: %v", removeErr)
				}
			}
		}

		// Record resize metric
		metrics.GetMetrics().RecordImageResize()

		// Generate URL
		sizeURL := s.generateURL(year, month, sizeFilename)

		results = append(results, ImageSize{
			Name:   sizeConfig.Name,
			Width:  resizedImg.Bounds().Dx(),
			Height: resizedImg.Bounds().Dy(),
			Path:   sizePath,
			URL:    sizeURL,
		})

		log.Printf("[ImageResize] Created %s: %dx%d at %s", sizeConfig.Name, resizedImg.Bounds().Dx(), resizedImg.Bounds().Dy(), sizePath)
	}

	return results, nil
}

// resizeImage resizes an image maintaining aspect ratio
func (s *ImageResizeService) resizeImage(ctx context.Context, img image.Image, maxWidth, maxHeight, origWidth, origHeight int) image.Image {
	// Validate inputs to prevent panic and DoS
	if maxWidth < 0 || maxHeight < 0 {
		log.Printf("[ImageResize] Invalid dimensions: maxWidth=%d, maxHeight=%d, returning original", maxWidth, maxHeight)
		return img
	}
	if origWidth <= 0 || origHeight <= 0 {
		log.Printf("[ImageResize] Invalid original dimensions: %dx%d, returning original", origWidth, origHeight)
		return img
	}

	// Limit maximum dimensions to prevent memory exhaustion
	const maxDimension = 10000
	if maxWidth > maxDimension {
		maxWidth = maxDimension
	}
	if maxHeight > maxDimension {
		maxHeight = maxDimension
	}

	// Handle cases where one dimension is 0 (no limit)
	var newWidth, newHeight int
	if maxHeight == 0 {
		// Only width limit (e.g., medium_large: 768x0)
		if origWidth <= maxWidth {
			return img // Don't upscale
		}
		newWidth = maxWidth
		newHeight = int(float64(origHeight) * float64(maxWidth) / float64(origWidth))
	} else if maxWidth == 0 {
		// Only height limit
		if origHeight <= maxHeight {
			return img // Don't upscale
		}
		newWidth = int(float64(origWidth) * float64(maxHeight) / float64(origHeight))
		newHeight = maxHeight
	} else {
		// Both dimensions specified - maintain aspect ratio
		// If original is smaller than target, don't upscale
		if origWidth <= maxWidth && origHeight <= maxHeight {
			return img
		}

		widthRatio := float64(maxWidth) / float64(origWidth)
		heightRatio := float64(maxHeight) / float64(origHeight)

		if widthRatio < heightRatio {
			newWidth = maxWidth
			newHeight = int(float64(origHeight) * widthRatio)
		} else {
			newWidth = int(float64(origWidth) * heightRatio)
			newHeight = maxHeight
		}
	}

	// Create new image with calculated dimensions
	return s.resizeBilinear(ctx, img, newWidth, newHeight)
}

// resizeBilinear performs bilinear interpolation resize for better quality
func (s *ImageResizeService) resizeBilinear(ctx context.Context, img image.Image, width, height int) image.Image {
	srcBounds := img.Bounds()
	srcWidth := srcBounds.Dx()
	srcHeight := srcBounds.Dy()

	// Create new RGBA image
	dst := image.NewRGBA(image.Rect(0, 0, width, height))

	// Validate dimensions to prevent division by zero
	if width <= 0 || height <= 0 {
		log.Printf("[ImageResize] Invalid resize dimensions: %dx%d, returning original", width, height)
		return img
	}
	if srcWidth <= 0 || srcHeight <= 0 {
		log.Printf("[ImageResize] Invalid source dimensions: %dx%d, returning original", srcWidth, srcHeight)
		return img
	}

	// Calculate ratios for bilinear interpolation
	// Use srcWidth-1 to map last pixel correctly (0-indexed to width-1)
	xRatio := float64(srcWidth-1) / float64(width)
	yRatio := float64(srcHeight-1) / float64(height)

	// Use worker pool for parallel processing to improve performance
	// Process rows in parallel using goroutines
	numWorkers := runtime.NumCPU()
	if numWorkers > height {
		numWorkers = height
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	// Use wait group to wait for all workers
	var wg sync.WaitGroup
	rowChan := make(chan int, height)

	// Start workers
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[ImageResize] Panic in resize worker: %v", r)
				}
			}()
			for y := range rowChan {
				// Check context cancellation periodically
				if y%100 == 0 {
					select {
					case <-ctx.Done():
						return
					default:
					}
				}

				// Process entire row
				for x := 0; x < width; x++ {
					// Map destination coordinates to source coordinates (floating point)
					srcX := xRatio * float64(x)
					srcY := yRatio * float64(y)

					// Get integer coordinates
					x1 := int(math.Floor(srcX))
					y1 := int(math.Floor(srcY))
					x2 := x1 + 1
					y2 := y1 + 1

					// Clamp to source bounds
					if x2 >= srcWidth {
						x2 = srcWidth - 1
					}
					if y2 >= srcHeight {
						y2 = srcHeight - 1
					}

					// Calculate fractional parts
					fx := srcX - float64(x1)
					fy := srcY - float64(y1)

					// Get four corner colors
					c11 := img.At(x1+srcBounds.Min.X, y1+srcBounds.Min.Y)
					c21 := img.At(x2+srcBounds.Min.X, y1+srcBounds.Min.Y)
					c12 := img.At(x1+srcBounds.Min.X, y2+srcBounds.Min.Y)
					c22 := img.At(x2+srcBounds.Min.X, y2+srcBounds.Min.Y)

					// Perform bilinear interpolation
					interpolated := bilinearInterpolate(c11, c21, c12, c22, fx, fy)
					dst.Set(x, y, interpolated)
				}
			}
		}()
	}

	// Send rows to workers
	go func() {
		defer close(rowChan)
		for y := 0; y < height; y++ {
			select {
			case <-ctx.Done():
				return
			case rowChan <- y:
			}
		}
	}()

	// Wait for all workers to complete
	wg.Wait()

	return dst
}

// bilinearInterpolate performs bilinear color interpolation
func bilinearInterpolate(c11, c21, c12, c22 color.Color, fx, fy float64) color.Color {
	// Convert colors to RGBA
	r11, g11, b11, a11 := c11.RGBA()
	r21, g21, b21, a21 := c21.RGBA()
	r12, g12, b12, a12 := c12.RGBA()
	r22, g22, b22, a22 := c22.RGBA()

	// Normalize to 0-255 range
	toFloat := func(v uint32) float64 {
		return float64(v>>8) / 255.0
	}

	// Interpolate horizontally first
	r1 := toFloat(r11)*(1-fx) + toFloat(r21)*fx
	g1 := toFloat(g11)*(1-fx) + toFloat(g21)*fx
	b1 := toFloat(b11)*(1-fx) + toFloat(b21)*fx
	a1 := toFloat(a11)*(1-fx) + toFloat(a21)*fx

	r2 := toFloat(r12)*(1-fx) + toFloat(r22)*fx
	g2 := toFloat(g12)*(1-fx) + toFloat(g22)*fx
	b2 := toFloat(b12)*(1-fx) + toFloat(b22)*fx
	a2 := toFloat(a12)*(1-fx) + toFloat(a22)*fx

	// Interpolate vertically
	r := r1*(1-fy) + r2*fy
	g := g1*(1-fy) + g2*fy
	b := b1*(1-fy) + b2*fy
	a := a1*(1-fy) + a2*fy

	// Convert back to uint8
	return color.RGBA{
		R: uint8(math.Round(r * 255)),
		G: uint8(math.Round(g * 255)),
		B: uint8(math.Round(b * 255)),
		A: uint8(math.Round(a * 255)),
	}
}

// saveImage saves image to file
func (s *ImageResizeService) saveImage(img image.Image, path string, format string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	switch strings.ToLower(format) {
	case "jpeg", "jpg":
		return jpeg.Encode(file, img, &jpeg.Options{Quality: DefaultJPEGQuality})
	case "png":
		return png.Encode(file, img)
	case "gif":
		return gif.Encode(file, img, nil)
	case "webp":
		// WebP encoding requires external library, fallback to JPEG
		log.Printf("[ImageResize] WebP encoding not fully supported, converting to JPEG")
		return jpeg.Encode(file, img, &jpeg.Options{Quality: DefaultJPEGQuality})
	default:
		// Default to JPEG
		return jpeg.Encode(file, img, &jpeg.Options{Quality: DefaultJPEGQuality})
	}
}

// generateSizeFilename generates filename for resized image
func (s *ImageResizeService) generateSizeFilename(baseName, ext, sizeName string) string {
	if sizeName == "full" || sizeName == "original" {
		return baseName + ext
	}
	return baseName + "-" + sizeName + ext
}

// generateURL generates URL for the file
func (s *ImageResizeService) generateURL(year, month, filename string) string {
	// Remove trailing slash from baseURL if present
	base := strings.TrimSuffix(s.baseURL, "/")
	return base + "/wp-content/uploads/" + year + "/" + month + "/" + filename
}

// SaveOriginal saves the original image to wp-uploads and optimizes it
func (s *ImageResizeService) SaveOriginal(ctx context.Context, sourcePath, filename string) (string, string, error) {
	// Check cancellation before starting
	select {
	case <-ctx.Done():
		return "", "", fmt.Errorf("save original cancelled: %w", ctx.Err())
	default:
	}

	year := utils.GetCurrentYear()
	month := utils.GetCurrentMonth()
	uploadPath := filepath.Join(s.wpUploadsDir, year, month)
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create upload directory %s: %w", uploadPath, err)
	}

	// Check cancellation after directory creation
	select {
	case <-ctx.Done():
		return "", "", fmt.Errorf("save original cancelled: %w", ctx.Err())
	default:
	}

	destPath := filepath.Join(uploadPath, filename)

	// Optimize original image if optimizer is available
	if s.optimizer != nil {
		optResult, err := s.optimizer.OptimizeImage(ctx, sourcePath, destPath)
		if err != nil {
			log.Printf("[ImageResize] Original optimization failed, using original: %v", err)
			// Fallback: copy original
			sourceFile, err := os.Open(sourcePath)
			if err != nil {
				return "", "", fmt.Errorf("failed to open source file %s: %w", sourcePath, err)
			}
			defer func() {
				if closeErr := sourceFile.Close(); closeErr != nil {
					log.Printf("[ImageResize] Failed to close source file: %v", closeErr)
				}
			}()

			destFile, err := os.Create(destPath)
			if err != nil {
				return "", "", fmt.Errorf("failed to create destination file %s: %w", destPath, err)
			}
			defer func() {
				if closeErr := destFile.Close(); closeErr != nil {
					log.Printf("[ImageResize] Failed to close destination file: %v", closeErr)
				}
			}()

			if _, err = io.Copy(destFile, sourceFile); err != nil {
				return "", "", fmt.Errorf("failed to copy file from %s to %s: %w", sourcePath, destPath, err)
			}
		} else {
			log.Printf("[ImageResize] Optimized original: saved %.2f%% (%d bytes)",
				optResult.SavedPercent, optResult.SavedBytes)
		}
	} else {
		// No optimizer, just copy
		sourceFile, err := os.Open(sourcePath)
		if err != nil {
			return "", "", fmt.Errorf("failed to open source file %s: %w", sourcePath, err)
		}
		defer func() {
			if closeErr := sourceFile.Close(); closeErr != nil {
				log.Printf("[ImageResize] Failed to close source file: %v", closeErr)
			}
		}()

		destFile, err := os.Create(destPath)
		if err != nil {
			return "", "", fmt.Errorf("failed to create destination file %s: %w", destPath, err)
		}
		defer func() {
			if closeErr := destFile.Close(); closeErr != nil {
				log.Printf("[ImageResize] Failed to close destination file: %v", closeErr)
			}
		}()

		if _, err = io.Copy(destFile, sourceFile); err != nil {
			return "", "", fmt.Errorf("failed to copy file from %s to %s: %w", sourcePath, destPath, err)
		}
	}

	url := s.generateURL(year, month, filename)
	return destPath, url, nil
}
