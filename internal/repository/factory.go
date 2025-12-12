package repository

import (
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type RepositoryType string

const (
	RepositoryTypeS3 RepositoryType = "s3"
)

type RepositoryFactory struct{}

func NewRepositoryFactory() *RepositoryFactory {
	return &RepositoryFactory{}
}

func (f *RepositoryFactory) CreateS3Repository(uploader *manager.Uploader, client *s3.Client, bucketName string) S3Repository {
	return NewS3Repository(uploader, client, bucketName)
}

func (f *RepositoryFactory) CreateRepository(repoType RepositoryType, uploader *manager.Uploader, client *s3.Client, bucketName string) (S3Repository, error) {
	switch repoType {
	case RepositoryTypeS3:
		return f.CreateS3Repository(uploader, client, bucketName), nil
	default:
		return nil, ErrUnsupportedRepositoryType
	}
}
