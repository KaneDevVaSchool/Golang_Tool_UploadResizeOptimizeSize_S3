package service

import (
	"time"

	"s3-upload-tool/internal/repository"
)

type ServiceType string

const (
	ServiceTypeUpload ServiceType = "upload"
)

type ServiceFactory struct {
	repoFactory *repository.RepositoryFactory
}

func NewServiceFactory(repoFactory *repository.RepositoryFactory) *ServiceFactory {
	return &ServiceFactory{
		repoFactory: repoFactory,
	}
}

func (f *ServiceFactory) CreateUploadService(
	s3Repo repository.S3Repository,
	bucketName, region, uploadDir string,
	maxSize int64,
	uploadTimeout time.Duration,
	useACL, usePresignedURL bool,
	presignedURLExpiry int,
	endpoint string,
	forcePathStyle bool,
	basePath string,
) UploadService {
	return NewUploadService(
		s3Repo,
		bucketName, region, uploadDir,
		maxSize, uploadTimeout,
		useACL, usePresignedURL, presignedURLExpiry,
		endpoint, forcePathStyle,
		basePath,
	)
}

func (f *ServiceFactory) CreateService(
	serviceType ServiceType,
	s3Repo repository.S3Repository,
	bucketName, region, uploadDir string,
	maxSize int64,
	uploadTimeout time.Duration,
	useACL, usePresignedURL bool,
	presignedURLExpiry int,
	endpoint string,
	forcePathStyle bool,
	basePath string,
) (UploadService, error) {
	switch serviceType {
	case ServiceTypeUpload:
		return f.CreateUploadService(
			s3Repo,
			bucketName, region, uploadDir,
			maxSize, uploadTimeout,
			useACL, usePresignedURL, presignedURLExpiry,
			endpoint, forcePathStyle,
			basePath,
		), nil
	default:
		return nil, ErrUnsupportedServiceType
	}
}
