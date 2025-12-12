package service

// Image processing constants
const (
	// DefaultJPEGQuality is the default quality for JPEG encoding
	DefaultJPEGQuality = 90

	// DefaultPNGQuality uses encoder default (0)
	DefaultPNGQuality = 0

	// File size thresholds for adaptive JPEG quality
	LargeFileThreshold  = 1024 * 1024 // 1MB
	MediumFileThreshold = 500 * 1024  // 500KB
	SmallFileThreshold  = 100 * 1024  // 100KB

	// Quality adjustment values
	LargeFileQualityReduction  = 15
	MediumFileQualityReduction = 10
	SmallFileQualityIncrease   = 5

	// JPEG quality bounds
	MinJPEGQuality = 70
	MaxJPEGQuality = 95

	// Default base quality when config is 0
	DefaultJPEGQualityBase = 85
)
