package service

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// ImageOptimizer xử lý image compression và optimization
type ImageOptimizer struct {
	jpegQuality int
	pngQuality  int
	enableWebP  bool
}

// OptimizationResult chứa optimization statistics
type OptimizationResult struct {
	OriginalSize  int64
	OptimizedSize int64
	SavedBytes    int64
	SavedPercent  float64
	Format        string
}

// NewImageOptimizer tạo image optimizer mới
func NewImageOptimizer(jpegQuality, pngQuality int, enableWebP bool) *ImageOptimizer {
	return &ImageOptimizer{
		jpegQuality: jpegQuality,
		pngQuality:  pngQuality,
		enableWebP:  enableWebP,
	}
}

// OptimizeImage optimize image file dùng native Go optimization
func (o *ImageOptimizer) OptimizeImage(ctx context.Context, inputPath, outputPath string) (*OptimizationResult, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("optimization cancelled: %w", ctx.Err())
	default:
	}

	originalInfo, err := os.Stat(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat input file %s: %w", inputPath, err)
	}
	originalSize := originalInfo.Size()

	ext := strings.ToLower(filepath.Ext(inputPath))
	format := strings.TrimPrefix(ext, ".")

	return o.optimizeNative(ctx, inputPath, outputPath, format, originalSize)
}

// optimizeNative dùng Go native libraries cho advanced optimization
func (o *ImageOptimizer) optimizeNative(ctx context.Context, inputPath, outputPath, format string, originalSize int64) (*OptimizationResult, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("optimization cancelled: %w", ctx.Err())
	default:
	}

	file, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open input file %s: %w", inputPath, err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image %s: %w", inputPath, err)
	}

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("optimization cancelled: %w", ctx.Err())
	default:
	}

	optimizedImg := o.stripMetadataAndOptimize(img)

	// * Thử nhiều quality levels để tìm compression tốt nhất
	var bestSize int64 = originalSize
	var bestPath string
	tempPaths := []string{}

	defer func() {
		for _, tempPath := range tempPaths {
			if tempPath != bestPath {
				if err := os.Remove(tempPath); err != nil {
					log.Printf("[ImageOptimizer] Failed to remove temp file %s: %v", tempPath, err)
				}
			}
		}
	}()

	switch format {
	case "jpg", "jpeg":
		// * Thử progressive encoding với nhiều quality levels
		baseQuality := o.calculateOptimalJPEGQuality(originalSize)
		qualities := []int{baseQuality - 5, baseQuality, baseQuality + 5}

		for i, quality := range qualities {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("optimization cancelled during JPEG encoding: %w", ctx.Err())
			default:
			}

			if quality < 60 || quality > 95 {
				continue
			}

			tempPath := fmt.Sprintf("%s.tmp%d", outputPath, i)
			tempPaths = append(tempPaths, tempPath)

			tempFile, err := os.Create(tempPath)
			if err != nil {
				continue
			}

			opts := &jpeg.Options{
				Quality: quality,
			}

			if err := jpeg.Encode(tempFile, optimizedImg, opts); err != nil {
				tempFile.Close()
				continue
			}
			tempFile.Close()

			fileInfo, err := os.Stat(tempPath)
			if err != nil {
				continue
			}

			if fileInfo.Size() < bestSize {
				bestSize = fileInfo.Size()
				bestPath = tempPath
			}
		}

		if bestPath == "" {
			bestPath = outputPath
			outputFile, err := os.Create(outputPath)
			if err != nil {
				return nil, err
			}
			if err := jpeg.Encode(outputFile, optimizedImg, &jpeg.Options{Quality: baseQuality}); err != nil {
				outputFile.Close()
				return nil, fmt.Errorf("failed to encode JPEG: %w", err)
			}
			outputFile.Close()
			bestSize, err = getFileSize(outputPath)
			if err != nil {
				return nil, fmt.Errorf("failed to get file size: %w", err)
			}
		} else {
			if err := os.Rename(bestPath, outputPath); err != nil {
				return nil, fmt.Errorf("failed to rename optimized file: %w", err)
			}
			bestSize, err = getFileSize(outputPath)
			if err != nil {
				return nil, fmt.Errorf("failed to get file size: %w", err)
			}
		}

	case "png":
		// * PNG optimization với nhiều compression levels
		compressionLevels := []png.CompressionLevel{
			png.DefaultCompression,
			png.BestSpeed,
			png.BestCompression,
		}

		for i, level := range compressionLevels {
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("optimization cancelled during PNG encoding: %w", ctx.Err())
			default:
			}

			tempPath := fmt.Sprintf("%s.tmp%d", outputPath, i)
			tempPaths = append(tempPaths, tempPath)

			tempFile, err := os.Create(tempPath)
			if err != nil {
				log.Printf("[ImageOptimizer] Failed to create temp file %s: %v", tempPath, err)
				continue
			}

			encoder := &png.Encoder{
				CompressionLevel: level,
			}

			if err := encoder.Encode(tempFile, optimizedImg); err != nil {
				tempFile.Close()
				log.Printf("[ImageOptimizer] Failed to encode PNG with compression level %d: %v", level, err)
				continue
			}
			if err := tempFile.Close(); err != nil {
				log.Printf("[ImageOptimizer] Failed to close temp file %s: %v", tempPath, err)
			}

			fileInfo, err := os.Stat(tempPath)
			if err != nil {
				log.Printf("[ImageOptimizer] Failed to stat temp file %s: %v", tempPath, err)
				continue
			}

			if fileInfo.Size() < bestSize {
				bestSize = fileInfo.Size()
				bestPath = tempPath
			}
		}

		if bestPath == "" {
			bestPath = outputPath
			outputFile, err := os.Create(outputPath)
			if err != nil {
				return nil, err
			}
			encoder := &png.Encoder{
				CompressionLevel: png.BestCompression,
			}
			if err := encoder.Encode(outputFile, optimizedImg); err != nil {
				outputFile.Close()
				return nil, fmt.Errorf("failed to encode PNG: %w", err)
			}
			outputFile.Close()
			bestSize, err = getFileSize(outputPath)
			if err != nil {
				return nil, fmt.Errorf("failed to get file size: %w", err)
			}
		} else {
			if err := os.Rename(bestPath, outputPath); err != nil {
				return nil, fmt.Errorf("failed to rename optimized file: %w", err)
			}
			bestSize, err = getFileSize(outputPath)
			if err != nil {
				return nil, fmt.Errorf("failed to get file size: %w", err)
			}
		}

	default:
		outputFile, err := os.Create(outputPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create output file: %w", err)
		}
		defer func() {
			if closeErr := outputFile.Close(); closeErr != nil {
				log.Printf("[ImageOptimizer] Failed to close output file: %v", closeErr)
			}
		}()

		if _, err := file.Seek(0, 0); err != nil {
			return nil, fmt.Errorf("failed to seek input file: %w", err)
		}

		bestSize, err = io.Copy(outputFile, file)
		if err != nil {
			return nil, fmt.Errorf("failed to copy file: %w", err)
		}

		if syncErr := outputFile.Sync(); syncErr != nil {
			log.Printf("[ImageOptimizer] Warning: failed to sync output file: %v", syncErr)
		}
	}

	savedBytes := originalSize - bestSize
	savedPercent := float64(savedBytes) / float64(originalSize) * 100

	log.Printf("[ImageOptimizer] Native Advanced: %d bytes -> %d bytes (saved %.2f%%)",
		originalSize, bestSize, savedPercent)

	return &OptimizationResult{
		OriginalSize:  originalSize,
		OptimizedSize: bestSize,
		SavedBytes:    savedBytes,
		SavedPercent:  savedPercent,
		Format:        format,
	}, nil
}

// stripMetadataAndOptimize xóa metadata và optimize color space
func (o *ImageOptimizer) stripMetadataAndOptimize(img image.Image) image.Image {
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	// * Convert to NRGBA nếu image có transparency để optimize color palette
	if _, ok := img.(*image.NRGBA); ok {
		nrgba := image.NewNRGBA(bounds)
		draw.Draw(nrgba, bounds, img, bounds.Min, draw.Src)
		return nrgba
	}

	// * Với JPEG, reduce color depth nhẹ để compression tốt hơn
	return o.reduceColorDepth(rgba)
}

// reduceColorDepth giảm color depth nhẹ để improve compression
func (o *ImageOptimizer) reduceColorDepth(img *image.RGBA) image.Image {
	bounds := img.Bounds()
	optimized := image.NewRGBA(bounds)

	// * Dùng worker pool cho parallel processing để improve performance
	numWorkers := runtime.NumCPU()
	height := bounds.Max.Y - bounds.Min.Y
	if numWorkers > height {
		numWorkers = height
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	var wg sync.WaitGroup
	rowChan := make(chan int, height)

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[ImageOptimizer] Panic in color depth reduction worker: %v", r)
				}
			}()
			for y := range rowChan {
				for x := bounds.Min.X; x < bounds.Max.X; x++ {
					c := img.RGBAAt(x, y)
					// * Slight quantization: round to nearest 4 để reduce color variations
					c.R = c.R &^ 3
					c.G = c.G &^ 3
					c.B = c.B &^ 3
					optimized.SetRGBA(x, y, c)
				}
			}
		}()
	}

	go func() {
		defer close(rowChan)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			rowChan <- y
		}
	}()

	wg.Wait()

	return optimized
}

func getFileSize(path string) (int64, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return fileInfo.Size(), nil
}

// calculateOptimalJPEGQuality tính optimal JPEG quality dựa trên file size
// * File lớn hơn → compression aggressive hơn
func (o *ImageOptimizer) calculateOptimalJPEGQuality(fileSize int64) int {
	baseQuality := o.jpegQuality
	if baseQuality == 0 {
		baseQuality = DefaultJPEGQualityBase
	}

	if fileSize > LargeFileThreshold {
		return max(MinJPEGQuality, baseQuality-LargeFileQualityReduction)
	} else if fileSize > MediumFileThreshold {
		return max(MinJPEGQuality+5, baseQuality-MediumFileQualityReduction)
	} else if fileSize < SmallFileThreshold {
		return min(MaxJPEGQuality, baseQuality+SmallFileQualityIncrease)
	}

	return baseQuality
}

// OptimizeImageInMemory optimize image từ memory
func (o *ImageOptimizer) OptimizeImageInMemory(img image.Image, format string, output io.Writer) (*OptimizationResult, error) {
	var buf bytes.Buffer

	switch format {
	case "jpg", "jpeg":
		quality := o.jpegQuality
		if quality == 0 {
			quality = 85
		}
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, err
		}
	case "png":
		encoder := &png.Encoder{
			CompressionLevel: png.BestCompression,
		}
		if err := encoder.Encode(&buf, img); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported format for optimization: %s", format)
	}

	optimizedSize := int64(buf.Len())

	_, err := io.Copy(output, &buf)
	if err != nil {
		return nil, err
	}

	return &OptimizationResult{
		OptimizedSize: optimizedSize,
		Format:        format,
	}, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
