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

// ParseS3ObjectKey trích S3 object key từ URL công khai (đối chiếu bucket).
// Trả chuỗi rỗng nếu URL không khớp dạng virtual-host, path-style, hoặc endpoint tuỳ chỉnh.
func ParseS3ObjectKey(objectURL, bucket string) string {
	objectURL = strings.TrimSpace(objectURL)
	bucket = strings.TrimSpace(bucket)
	if objectURL == "" || bucket == "" {
		return ""
	}

	u, err := url.Parse(objectURL)
	if err != nil {
		return ""
	}

	path := strings.TrimPrefix(u.Path, "/")
	if path == "" {
		return ""
	}

	if i := strings.Index(path, "/"); i >= 0 && path[:i] == bucket {
		return unescapeS3KeySegments(path[i+1:])
	}

	host := strings.ToLower(u.Hostname())
	bucketHostPrefix := strings.ToLower(bucket) + ".s3"
	if strings.HasPrefix(host, bucketHostPrefix) {
		return unescapeS3KeySegments(path)
	}

	return ""
}

func unescapeS3KeySegments(keyPath string) string {
	parts := strings.Split(keyPath, "/")
	for i, p := range parts {
		decoded, err := url.PathUnescape(p)
		if err == nil {
			parts[i] = decoded
		}
	}
	return strings.Join(parts, "/")
}
