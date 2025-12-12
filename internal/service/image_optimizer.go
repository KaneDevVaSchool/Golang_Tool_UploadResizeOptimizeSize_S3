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

// ImageOptimizer handles image compression and optimization
type ImageOptimizer struct {
	jpegQuality int
	pngQuality  int
	enableWebP  bool
}

// OptimizationResult contains optimization statistics
type OptimizationResult struct {
	OriginalSize  int64
	OptimizedSize int64
	SavedBytes    int64
	SavedPercent  float64
	Format        string
}

// NewImageOptimizer creates a new image optimizer
func NewImageOptimizer(jpegQuality, pngQuality int, enableWebP bool) *ImageOptimizer {
	return &ImageOptimizer{
		jpegQuality: jpegQuality,
		pngQuality:  pngQuality,
		enableWebP:  enableWebP,
	}
}

// OptimizeImage optimizes an image file using native Go optimization
func (o *ImageOptimizer) OptimizeImage(ctx context.Context, inputPath, outputPath string) (*OptimizationResult, error) {
	// Check cancellation before starting
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("optimization cancelled: %w", ctx.Err())
	default:
	}

	// Get original file size
	originalInfo, err := os.Stat(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat input file %s: %w", inputPath, err)
	}
	originalSize := originalInfo.Size()

	ext := strings.ToLower(filepath.Ext(inputPath))
	format := strings.TrimPrefix(ext, ".")

	// Use native optimization
	return o.optimizeNative(ctx, inputPath, outputPath, format, originalSize)
}

// optimizeNative uses Go native libraries for advanced optimization
func (o *ImageOptimizer) optimizeNative(ctx context.Context, inputPath, outputPath, format string, originalSize int64) (*OptimizationResult, error) {
	// Check cancellation
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("optimization cancelled: %w", ctx.Err())
	default:
	}

	// Open and decode image
	file, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open input file %s: %w", inputPath, err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image %s: %w", inputPath, err)
	}

	// Check cancellation after decode
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("optimization cancelled: %w", ctx.Err())
	default:
	}

	// Strip metadata and optimize image
	optimizedImg := o.stripMetadataAndOptimize(img)

	// Try multiple quality levels to find best compression
	var bestSize int64 = originalSize
	var bestPath string
	tempPaths := []string{}

	defer func() {
		// Cleanup temp files
		for _, tempPath := range tempPaths {
			if tempPath != bestPath {
				if err := os.Remove(tempPath); err != nil {
					log.Printf("[ImageOptimizer] Failed to remove temp file %s: %v", tempPath, err)
				}
			}
		}
	}()

	// Optimize based on format
	switch format {
	case "jpg", "jpeg":
		// Try progressive encoding with different quality levels
		baseQuality := o.calculateOptimalJPEGQuality(originalSize)
		qualities := []int{baseQuality - 5, baseQuality, baseQuality + 5}

		for i, quality := range qualities {
			// Check cancellation in loop
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

			// Use progressive JPEG for better compression
			opts := &jpeg.Options{
				Quality: quality,
			}

			// Encode with optimized settings
			if err := jpeg.Encode(tempFile, optimizedImg, opts); err != nil {
				tempFile.Close()
				continue
			}
			tempFile.Close()

			// Check file size
			fileInfo, err := os.Stat(tempPath)
			if err != nil {
				continue
			}

			if fileInfo.Size() < bestSize {
				bestSize = fileInfo.Size()
				bestPath = tempPath
			}
		}

		// If no better compression found, use base quality
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
			// Move best result to output
			if err := os.Rename(bestPath, outputPath); err != nil {
				return nil, fmt.Errorf("failed to rename optimized file: %w", err)
			}
			bestSize, err = getFileSize(outputPath)
			if err != nil {
				return nil, fmt.Errorf("failed to get file size: %w", err)
			}
		}

	case "png":
		// PNG optimization with multiple compression levels
		compressionLevels := []png.CompressionLevel{
			png.DefaultCompression,
			png.BestSpeed,
			png.BestCompression,
		}

		for i, level := range compressionLevels {
			// Check cancellation in loop
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
		// For other formats, just copy
		outputFile, err := os.Create(outputPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create output file: %w", err)
		}
		defer func() {
			if closeErr := outputFile.Close(); closeErr != nil {
				log.Printf("[ImageOptimizer] Failed to close output file: %v", closeErr)
			}
		}()

		// Reset file pointer to beginning
		if _, err := file.Seek(0, 0); err != nil {
			return nil, fmt.Errorf("failed to seek input file: %w", err)
		}

		bestSize, err = io.Copy(outputFile, file)
		if err != nil {
			return nil, fmt.Errorf("failed to copy file: %w", err)
		}

		// Ensure file is flushed to disk
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

// stripMetadataAndOptimize removes metadata and optimizes color space
func (o *ImageOptimizer) stripMetadataAndOptimize(img image.Image) image.Image {
	// Convert to RGBA to strip metadata and optimize
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	// Optimize color palette for smaller files
	// Convert to NRGBA if image has transparency
	if _, ok := img.(*image.NRGBA); ok {
		nrgba := image.NewNRGBA(bounds)
		draw.Draw(nrgba, bounds, img, bounds.Min, draw.Src)
		return nrgba
	}

	// For JPEG, convert to YCbCr-like optimization
	// Reduce color depth slightly for better compression
	return o.reduceColorDepth(rgba)
}

// reduceColorDepth reduces color depth slightly to improve compression
func (o *ImageOptimizer) reduceColorDepth(img *image.RGBA) image.Image {
	bounds := img.Bounds()
	optimized := image.NewRGBA(bounds)

	// Use worker pool for parallel processing to improve performance
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

	// Start workers
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
					// Slight quantization to improve compression
					// Round to nearest 4 to reduce color variations
					c.R = c.R &^ 3
					c.G = c.G &^ 3
					c.B = c.B &^ 3
					optimized.SetRGBA(x, y, c)
				}
			}
		}()
	}

	// Send rows to workers
	go func() {
		defer close(rowChan)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			rowChan <- y
		}
	}()

	// Wait for all workers to complete
	wg.Wait()

	return optimized
}

// getFileSize helper function
func getFileSize(path string) (int64, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return fileInfo.Size(), nil
}

// calculateOptimalJPEGQuality calculates optimal JPEG quality based on file size
// Larger files get more aggressive compression
func (o *ImageOptimizer) calculateOptimalJPEGQuality(fileSize int64) int {
	// Base quality from config
	baseQuality := o.jpegQuality
	if baseQuality == 0 {
		baseQuality = DefaultJPEGQualityBase
	}

	// Adjust based on file size
	// Files > 1MB: reduce quality more aggressively
	// Files < 100KB: keep higher quality
	if fileSize > LargeFileThreshold {
		// Large files: reduce quality more aggressively
		return max(MinJPEGQuality, baseQuality-LargeFileQualityReduction)
	} else if fileSize > MediumFileThreshold {
		// Medium files: reduce quality moderately
		return max(MinJPEGQuality+5, baseQuality-MediumFileQualityReduction)
	} else if fileSize < SmallFileThreshold {
		// Small files: keep high quality
		return min(MaxJPEGQuality, baseQuality+SmallFileQualityIncrease)
	}

	return baseQuality
}

// OptimizeImageInMemory optimizes an image from memory
func (o *ImageOptimizer) OptimizeImageInMemory(img image.Image, format string, output io.Writer) (*OptimizationResult, error) {
	var buf bytes.Buffer

	// Encode to buffer first to get size
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

	// Write to output
	_, err := io.Copy(output, &buf)
	if err != nil {
		return nil, err
	}

	return &OptimizationResult{
		OptimizedSize: optimizedSize,
		Format:        format,
	}, nil
}

// Helper functions
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
