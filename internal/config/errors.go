package config

import "errors"

var (
	ErrMissingBucketName = errors.New("S3_BUCKET_NAME environment variable is required")
)
