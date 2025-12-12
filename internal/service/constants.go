package service

// Constants cho image processing
const (
	DefaultJPEGQuality = 90
	DefaultPNGQuality  = 0 // * Encoder default

	// * File size thresholds cho adaptive JPEG quality
	LargeFileThreshold  = 1024 * 1024
	MediumFileThreshold = 500 * 1024
	SmallFileThreshold  = 100 * 1024

	// * Quality adjustment values
	LargeFileQualityReduction  = 15
	MediumFileQualityReduction = 10
	SmallFileQualityIncrease   = 5

	// * JPEG quality bounds
	MinJPEGQuality = 70
	MaxJPEGQuality = 95

	DefaultJPEGQualityBase = 85 // * Base quality khi config = 0
)
