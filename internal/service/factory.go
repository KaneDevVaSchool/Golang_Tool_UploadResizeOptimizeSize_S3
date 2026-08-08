package service

import (
	"time"

	"s3-upload-tool/internal/database"
	"s3-upload-tool/internal/repository"
)

type ServiceType string

const (
	ServiceTypeUpload ServiceType = "upload"
)

type ServiceFactory struct {
	repoFactory *repository.RepositoryFactory
	db          *database.DB
}

func NewServiceFactory(repoFactory *repository.RepositoryFactory, db *database.DB) *ServiceFactory {
	return &ServiceFactory{
		repoFactory: repoFactory,
		db:          db,
	}
}

func (f *ServiceFactory) CreateUploadService(
	s3Repo repository.S3Repository,
	uploadRepo repository.UploadRepository,
	bucketName, region, uploadDir string,
	maxSize int64,
	uploadTimeout time.Duration,
	useACL, usePresignedURL bool,
	presignedURLExpiry int,
	endpoint string,
	forcePathStyle bool,
) UploadService {
	return NewUploadService(
		s3Repo, uploadRepo, f.db,
		bucketName, region, uploadDir,
		maxSize, uploadTimeout,
		useACL, usePresignedURL, presignedURLExpiry,
		endpoint, forcePathStyle,
	)
}

func (f *ServiceFactory) CreateService(
	serviceType ServiceType,
	s3Repo repository.S3Repository,
	uploadRepo repository.UploadRepository,
	bucketName, region, uploadDir string,
	maxSize int64,
	uploadTimeout time.Duration,
	useACL, usePresignedURL bool,
	presignedURLExpiry int,
	endpoint string,
	forcePathStyle bool,
) (UploadService, error) {
	switch serviceType {
	case ServiceTypeUpload:
		return f.CreateUploadService(
			s3Repo, uploadRepo,
			bucketName, region, uploadDir,
			maxSize, uploadTimeout,
			useACL, usePresignedURL, presignedURLExpiry,
			endpoint, forcePathStyle,
		), nil
	default:
		return nil, ErrUnsupportedServiceType
	}
}
