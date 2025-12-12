package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"s3-upload-tool/internal/database"
	"s3-upload-tool/internal/models"
	"s3-upload-tool/internal/repository"
	"s3-upload-tool/internal/utils"
)

type UploadService interface {
	UploadImage(ctx context.Context, filename string, file io.Reader, fileSize int64, maxSize int64) (*models.UploadResponse, error)
	UploadImageWithTransaction(ctx context.Context, filename string, file io.Reader, fileSize int64, maxSize int64) (*models.UploadResponse, *models.UploadRecord, error)
}

type uploadService struct {
	s3Repo             repository.S3Repository
	uploadRepo         repository.UploadRepository
	db                 *database.DB
	bucketName         string
	region             string
	uploadDir          string
	maxSize            int64
	uploadTimeout      time.Duration
	useACL             bool
	usePresignedURL    bool
	presignedURLExpiry int
}

func NewUploadService(s3Repo repository.S3Repository, uploadRepo repository.UploadRepository, db *database.DB, bucketName, region, uploadDir string, maxSize int64, uploadTimeout time.Duration, useACL bool, usePresignedURL bool, presignedURLExpiry int) UploadService {
	return &uploadService{
		s3Repo:             s3Repo,
		uploadRepo:         uploadRepo,
		db:                 db,
		bucketName:         bucketName,
		region:             region,
		uploadDir:          uploadDir,
		maxSize:            maxSize,
		uploadTimeout:      uploadTimeout,
		useACL:             useACL,
		usePresignedURL:    usePresignedURL,
		presignedURLExpiry: presignedURLExpiry,
	}
}

func (s *uploadService) UploadImage(ctx context.Context, filename string, file io.Reader, fileSize int64, maxSize int64) (*models.UploadResponse, error) {
	uploadCtx := ctx
	if s.uploadTimeout > 0 {
		var cancel context.CancelFunc
		uploadCtx, cancel = context.WithTimeout(ctx, s.uploadTimeout)
		defer cancel()
	}

	log.Printf("[UploadService] Bắt đầu upload file: %s, kích thước: %s", filename, utils.FormatFileSize(fileSize))

	if !utils.IsAllowedFileType(filename) {
		log.Printf("[UploadService] File type không được hỗ trợ: %s", filename)
		return nil, ErrInvalidFileFormat
	}

	if err := utils.ValidateFileSizeFromHeader(fileSize, maxSize); err != nil {
		log.Printf("[UploadService] File size validation failed: %v", err)
		return nil, fmt.Errorf("file size validation failed: %w", NewFileSizeError(fileSize, maxSize, err.Error()))
	}

	log.Printf("[UploadService] Tạo file tạm trong: %s", s.uploadDir)
	tempFile, err := os.CreateTemp(s.uploadDir, "upload-*"+filepath.Ext(filename))
	if err != nil {
		log.Printf("[UploadService] Failed to create temp file: %v", err)
		return nil, fmt.Errorf("failed to create temp file in %s: %w", s.uploadDir, ErrCreateTempFile)
	}
	tempFileName := tempFile.Name()
	log.Printf("[UploadService] File tạm đã tạo: %s", tempFileName)
	defer func() {
		if removeErr := os.Remove(tempFileName); removeErr != nil {
			log.Printf("[UploadService] Cảnh báo: Không thể xóa file tạm %s: %v", tempFileName, removeErr)
		} else {
			log.Printf("[UploadService] Đã xóa file tạm: %s", tempFileName)
		}
	}()
	defer tempFile.Close()

	log.Printf("[UploadService] Đang copy dữ liệu vào file tạm...")
	// ? Dùng maxSize+1 để phát hiện file vượt limit, nhưng validate chặt chẽ
	limitedReader := io.LimitReader(file, maxSize+1)

	// ! Timeout 30s cho file copy để tránh treo
	copyCtx, copyCancel := context.WithTimeout(uploadCtx, 30*time.Second)
	defer copyCancel()

	// ? io.Copy không hỗ trợ context cancellation, nên đóng file để interrupt
	done := make(chan error, 1)
	var bytesWritten int64
	copyDone := make(chan struct{})

	go func() {
		defer close(copyDone)
		var copyErr error
		bytesWritten, copyErr = io.Copy(tempFile, limitedReader)
		done <- copyErr
	}()

	select {
	case err := <-done:
		if err != nil {
			log.Printf("[UploadService] Lỗi khi copy file: %v", err)
			return nil, fmt.Errorf("failed to copy file: %w", ErrSaveFile)
		}
	case <-copyCtx.Done():
		log.Printf("[UploadService] File copy timeout: %v", copyCtx.Err())
		// ! Đóng file để interrupt I/O, goroutine sẽ exit khi io.Copy trả về error
		if closeErr := tempFile.Close(); closeErr != nil {
			log.Printf("[UploadService] Error closing temp file on timeout: %v", closeErr)
		}

		// ! Đợi goroutine exit với timeout 3s
		waitCtx, waitCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer waitCancel()

		select {
		case <-copyDone:
			log.Printf("[UploadService] Copy goroutine exited cleanly after timeout")
		case <-waitCtx.Done():
			log.Printf("[UploadService] Warning: copy goroutine did not exit within 3 seconds after timeout")
			// ! Đã làm hết sức, goroutine sẽ exit khi write vào file/channel đã đóng
		}

		select {
		case err := <-done:
			if err != nil {
				log.Printf("[UploadService] Copy error after timeout: %v", err)
			}
		default:
		}

		return nil, fmt.Errorf("file copy timeout: %w", copyCtx.Err())
	}

	// ! Validate chặt: file phải <= maxSize
	if bytesWritten > maxSize {
		log.Printf("[UploadService] File vượt quá giới hạn: %d bytes > %d bytes", bytesWritten, maxSize)
		return nil, NewFileSizeError(bytesWritten, maxSize,
			fmt.Sprintf("file quá lớn: %s (giới hạn: %s)",
				utils.FormatFileSize(bytesWritten), utils.FormatFileSize(maxSize)))
	}
	// ! Kiểm tra nếu đọc đúng maxSize+1 (file bị truncate)
	if bytesWritten == maxSize+1 {
		log.Printf("[UploadService] File vượt quá giới hạn (truncated): %d bytes >= %d bytes", bytesWritten, maxSize)
		return nil, NewFileSizeError(bytesWritten, maxSize,
			fmt.Sprintf("file quá lớn: vượt quá giới hạn %s",
				utils.FormatFileSize(maxSize)))
	}

	log.Printf("[UploadService] Đã copy %d bytes (%s) vào file tạm", bytesWritten, utils.FormatFileSize(bytesWritten))

	if _, err := tempFile.Seek(0, 0); err != nil {
		log.Printf("[UploadService] Failed to seek file: %v", err)
		return nil, fmt.Errorf("failed to seek temp file: %w", ErrSaveFile)
	}

	key := utils.GenerateS3Key(filename)
	contentType := utils.GetContentType(filename)
	log.Printf("[UploadService] S3 Key: %s, Content-Type: %s", key, contentType)

	log.Printf("[UploadService] Đang upload lên S3 bucket: %s (UseACL: %v, UsePresignedURL: %v)", s.bucketName, s.useACL, s.usePresignedURL)
	_, uploadErr := s.s3Repo.Upload(uploadCtx, s.bucketName, key, tempFile, contentType, s.useACL)
	if uploadErr != nil {
		log.Printf("[UploadService] Failed to upload to S3: %v", uploadErr)
		return nil, fmt.Errorf("failed to upload to S3 bucket %s, key %s: %w", s.bucketName, key, ErrUploadToS3)
	}

	var url string
	if s.usePresignedURL {
		expiry := time.Duration(s.presignedURLExpiry) * time.Minute
		if expiry == 0 {
			expiry = 60 * time.Minute // Default 1 hour
		}
		log.Printf("[UploadService] Tạo pre-signed URL với expiry: %v", expiry)
		presignedURL, err := s.s3Repo.GeneratePresignedURL(uploadCtx, s.bucketName, key, expiry)
		if err != nil {
			log.Printf("[UploadService] Failed to generate presigned URL: %v", err)
			return nil, fmt.Errorf("failed to generate presigned URL for bucket %s, key %s: %w", s.bucketName, key, ErrUploadToS3)
		}
		url = presignedURL
		log.Printf("[UploadService] Upload thành công! Pre-signed URL: %s", url)
	} else {
		url = fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucketName, s.region, key)
		log.Printf("[UploadService] Upload thành công! URL: %s", url)
	}

	return &models.UploadResponse{
		URL: url,
		Key: key,
	}, nil
}

// UploadImageWithTransaction uploads image with database transaction support
func (s *uploadService) UploadImageWithTransaction(ctx context.Context, filename string, file io.Reader, fileSize int64, maxSize int64) (resp *models.UploadResponse, record *models.UploadRecord, err error) {
	// Start database transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// ! Dùng named return variables để đảm bảo rollback đúng
	var txErr error
	defer func() {
		// ! Xử lý panic: rollback trước khi re-panic
		if p := recover(); p != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("[UploadService] Failed to rollback on panic: %v", rollbackErr)
			}
			panic(p)
		}

		// ! Rollback nếu có bất kỳ lỗi nào
		if err != nil || txErr != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("[UploadService] Failed to rollback transaction: %v (original error: %v)", rollbackErr, err)
			}
		}
	}()

	uploadRecord := &models.UploadRecord{
		Filename:     filename,
		OriginalName: filename,
		FileSize:     fileSize,
		ContentType:  utils.GetContentType(filename),
		Status:       string(models.UploadStatusPending),
	}

	record, err = s.uploadRepo.CreateUpload(ctx, tx, uploadRecord)
	if err != nil {
		txErr = err
		return nil, nil, fmt.Errorf("failed to create upload record: %w", err)
	}

	uploadCtx := ctx
	if s.uploadTimeout > 0 {
		var cancel context.CancelFunc
		uploadCtx, cancel = context.WithTimeout(ctx, s.uploadTimeout)
		defer cancel()
	}

	log.Printf("[UploadService] Starting upload with transaction: record_id=%d, filename=%s", record.ID, filename)

	key := utils.GenerateS3Key(filename)
	contentType := utils.GetContentType(filename)

	tempFile, err := os.CreateTemp(s.uploadDir, "upload-*"+filepath.Ext(filename))
	if err != nil {
		txErr = err
		s.uploadRepo.UpdateUploadStatus(ctx, tx, record.ID, models.UploadStatusFailed, "", "", err)
		return nil, record, fmt.Errorf("failed to create temp file: %w", err)
	}
	tempFileName := tempFile.Name()
	defer func() {
		tempFile.Close()
		if removeErr := os.Remove(tempFileName); removeErr != nil {
			log.Printf("[UploadService] Warning: Failed to remove temp file %s: %v", tempFileName, removeErr)
		}
	}()

	limitedReader := io.LimitReader(file, maxSize+1)
	bytesWritten, err := io.Copy(tempFile, limitedReader)
	if err != nil {
		txErr = err
		s.uploadRepo.UpdateUploadStatus(ctx, tx, record.ID, models.UploadStatusFailed, "", "", err)
		return nil, record, fmt.Errorf("failed to copy file: %w", err)
	}

	if bytesWritten > maxSize {
		err = NewFileSizeError(bytesWritten, maxSize, fmt.Sprintf("file quá lớn: %s (giới hạn: %s)", utils.FormatFileSize(bytesWritten), utils.FormatFileSize(maxSize)))
		txErr = err
		s.uploadRepo.UpdateUploadStatus(ctx, tx, record.ID, models.UploadStatusFailed, "", "", err)
		return nil, record, err
	}

	if _, err := tempFile.Seek(0, 0); err != nil {
		txErr = err
		s.uploadRepo.UpdateUploadStatus(ctx, tx, record.ID, models.UploadStatusFailed, "", "", err)
		return nil, record, fmt.Errorf("failed to seek file: %w", err)
	}

	_, uploadErr := s.s3Repo.Upload(uploadCtx, s.bucketName, key, tempFile, contentType, s.useACL)
	if uploadErr != nil {
		txErr = uploadErr
		s.uploadRepo.UpdateUploadStatus(ctx, tx, record.ID, models.UploadStatusFailed, key, "", uploadErr)
		return nil, record, fmt.Errorf("failed to upload to S3: %w", uploadErr)
	}

	var url string
	if s.usePresignedURL {
		expiry := time.Duration(s.presignedURLExpiry) * time.Minute
		if expiry == 0 {
			expiry = 60 * time.Minute
		}
		presignedURL, err := s.s3Repo.GeneratePresignedURL(uploadCtx, s.bucketName, key, expiry)
		if err != nil {
			txErr = err
			s.uploadRepo.UpdateUploadStatus(ctx, tx, record.ID, models.UploadStatusFailed, key, "", err)
			return nil, record, fmt.Errorf("failed to generate presigned URL: %w", err)
		}
		url = presignedURL
	} else {
		url = fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucketName, s.region, key)
	}

	if err = s.uploadRepo.UpdateUploadStatus(ctx, tx, record.ID, models.UploadStatusCompleted, key, url, nil); err != nil {
		txErr = err
		return nil, record, fmt.Errorf("failed to update upload status: %w", err)
	}

	if err = tx.Commit(); err != nil {
		txErr = err
		return nil, record, fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("[UploadService] Upload completed successfully with transaction: record_id=%d, url=%s", record.ID, url)

	resp = &models.UploadResponse{
		URL: url,
		Key: key,
	}
	return resp, record, nil
}
