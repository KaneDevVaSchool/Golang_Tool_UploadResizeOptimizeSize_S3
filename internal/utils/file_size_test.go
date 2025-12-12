package utils

import (
	"testing"
)

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"bytes", 512, "512 B"},
		{"kilobytes", 1024, "1.00 KB"},
		{"megabytes", 1024 * 1024, "1.00 MB"},
		{"large megabytes", 5 * 1024 * 1024, "5.00 MB"},
		{"zero", 0, "0 B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatFileSize(tt.bytes)
			if got != tt.expected {
				t.Errorf("FormatFileSize(%d) = %v, want %v", tt.bytes, got, tt.expected)
			}
		})
	}
}

func TestValidateFileSizeFromHeader(t *testing.T) {
	tests := []struct {
		name    string
		size    int64
		maxSize int64
		wantErr bool
	}{
		{"valid size", 1024, 2048, false},
		{"exact max", 2048, 2048, false},
		{"too large", 3000, 2048, true},
		{"zero size", 0, 2048, true},
		{"negative size", -1, 2048, false}, // Negative size passes validation (handled by other checks)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFileSizeFromHeader(tt.size, tt.maxSize)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFileSizeFromHeader() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
