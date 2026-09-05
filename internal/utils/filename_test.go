package utils

import (
	"strings"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantErr   bool
		checkFunc func(string) bool
	}{
		{
			name:    "normal filename",
			input:   "test.jpg",
			wantErr: false,
			checkFunc: func(s string) bool {
				return s == "test.jpg"
			},
		},
		{
			name:    "path traversal attempt",
			input:   "../../../etc/passwd",
			wantErr: false,
			checkFunc: func(s string) bool {
				return s != "" && s != "../../../etc/passwd" && !strings.Contains(s, "..")
			},
		},
		{
			name:    "empty filename",
			input:   "",
			wantErr: true,
		},
		{
			name:    "filename with dangerous characters",
			input:   "file<>name.jpg",
			wantErr: false,
			checkFunc: func(s string) bool {
				return !strings.Contains(s, "<") && !strings.Contains(s, ">")
			},
		},
		{
			name:    "filename with path separators",
			input:   "path/to/file.jpg",
			wantErr: false,
			checkFunc: func(s string) bool {
				return !strings.Contains(s, "/") && !strings.Contains(s, "\\")
			},
		},
		{
			name:    "very long filename",
			input:   string(make([]byte, 300)) + ".jpg",
			wantErr: false,
			checkFunc: func(s string) bool {
				return len(s) <= MaxFilenameLength
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SanitizeFilename(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("SanitizeFilename() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.checkFunc != nil {
				if !tt.checkFunc(got) {
					t.Errorf("SanitizeFilename() = %v, failed check", got)
				}
			}
		})
	}
}
