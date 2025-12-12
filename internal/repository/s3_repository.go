package repository

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Repository interface {
	Upload(ctx context.Context, bucket, key string, body io.Reader, contentType string, useACL bool) (string, error)
	GeneratePresignedURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error)
	CheckConnectivity(ctx context.Context) error
}

type s3Repository struct {
	uploader   *manager.Uploader
	client     *s3.Client
	bucketName string
}

func NewS3Repository(uploader *manager.Uploader, client *s3.Client, bucketName string) S3Repository {
	return &s3Repository{
		uploader:   uploader,
		client:     client,
		bucketName: bucketName,
	}
}

func (r *s3Repository) Upload(ctx context.Context, bucket, key string, body io.Reader, contentType string, useACL bool) (string, error) {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	}

	if useACL {
		input.ACL = types.ObjectCannedACLPublicRead
	}

	result, err := r.uploader.Upload(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3 bucket %s, key %s: %w", bucket, key, err)
	}

	return result.Location, nil
}

func (r *s3Repository) GeneratePresignedURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	presigner := s3.NewPresignClient(r.client)

	request, err := presigner.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = expiry
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL for bucket %s, key %s: %w", bucket, key, err)
	}

	return request.URL, nil
}

// CheckConnectivity tests S3 connectivity by performing a HeadBucket operation
func (r *s3Repository) CheckConnectivity(ctx context.Context) error {
	_, err := r.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(r.bucketName),
	})
	if err != nil {
		return fmt.Errorf("S3 connectivity check failed for bucket %s: %w", r.bucketName, err)
	}
	return nil
}
