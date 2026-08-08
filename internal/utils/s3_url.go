package utils

import (
	"fmt"
	"net/url"
	"strings"
)

// BuildS3ObjectURL builds a public (or endpoint-relative) object URL.
// Prefer presigned URLs when the bucket is private; this is the non-presigned fallback.
func BuildS3ObjectURL(bucket, region, key, endpoint string, forcePathStyle bool) string {
	key = strings.TrimPrefix(key, "/")
	escapedKey := escapeS3Key(key)

	if endpoint != "" {
		base := strings.TrimRight(endpoint, "/")
		// Custom endpoints (MinIO / LocalStack) use path-style URLs.
		return fmt.Sprintf("%s/%s/%s", base, bucket, escapedKey)
	}

	if region == "" {
		region = "us-east-1"
	}
	if forcePathStyle {
		return fmt.Sprintf("https://s3.%s.amazonaws.com/%s/%s", region, bucket, escapedKey)
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, region, escapedKey)
}

func escapeS3Key(key string) string {
	parts := strings.Split(key, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}
