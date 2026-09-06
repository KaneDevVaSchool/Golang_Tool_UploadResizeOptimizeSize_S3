package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"s3-upload-tool/internal/models"
	"s3-upload-tool/internal/repository"
	"s3-upload-tool/internal/utils"
)

type UploadService interface {
	UploadImage(ctx context.Context, filename string, file io.Reader, fileSize int64, maxSize int64) (*models.UploadResponse, error)
	// UploadDerived đẩy một file dẫn xuất (biến thể ảnh đã resize) lên S3 tại
	// key cho trước và trả URL công khai.
	//
	// Khác UploadImage ở chỗ: không sinh key mới, không kiểm tra kích thước,
	// không ghi bảng uploads - dữ liệu đã nằm sẵn trong RAM và do server tạo
	// ra chứ không phải người dùng gửi lên, nên mọi bước xác thực đầu vào ở
	// đó đều thừa. Key do phía gọi đặt để bám theo key của ảnh gốc.
	UploadDerived(ctx context.Context, key string, data []byte, contentType string) (string, error)
	// ObjectKeyFromURL trích S3 key từ URL object (variants, thumbnail).
	ObjectKeyFromURL(objectURL string) string
	// DeleteObject xoá một object khỏi bucket đã cấu hình.
	DeleteObject(ctx context.Context, key string) error
}

type uploadService struct {
	s3Repo             repository.S3Repository
	bucketName         string
	basePath           string
	region             string
	endpoint           string
	forcePathStyle     bool
	uploadDir          string
	maxSize            int64
	uploadTimeout      time.Duration
	useACL             bool
	usePresignedURL    bool
	presignedURLExpiry int
}

func NewUploadService(s3Repo repository.S3Repository, bucketName, region, uploadDir string, maxSize int64, uploadTimeout time.Duration, useACL bool, usePresignedURL bool, presignedURLExpiry int, endpoint string, forcePathStyle bool, basePath string) UploadService {
	return &uploadService{
		s3Repo:             s3Repo,
		bucketName:         bucketName,
		basePath:           basePath,
		region:             region,
		endpoint:           endpoint,
		forcePathStyle:     forcePathStyle,
		uploadDir:          uploadDir,
		maxSize:            maxSize,
		uploadTimeout:      uploadTimeout,
		useACL:             useACL,
		usePresignedURL:    usePresignedURL,
		presignedURLExpiry: presignedURLExpiry,
	}
}

func (s *uploadService) objectURL(key string) string {
	return utils.BuildS3ObjectURL(s.bucketName, s.region, key, s.endpoint, s.forcePathStyle)
}

func (s *uploadService) UploadDerived(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	if len(data) == 0 {
		return "", fmt.Errorf("dữ liệu biến thể rỗng cho key %s", key)
	}

	if _, err := s.s3Repo.Upload(ctx, s.bucketName, key, bytes.NewReader(data), contentType, s.useACL); err != nil {
		return "", fmt.Errorf("failed to upload derived object %s: %w", key, err)
	}

	// Luôn dùng URL tĩnh, kể cả khi usePresignedURL đang bật: URL của biến
	// thể được lưu vào DB y như s3_url, mà presigned URL thì hết hạn - đúng
	// cái bẫy đã khiến ảnh trả 403 sau vài giờ (xem docs/S3-PUBLIC-READ.md).
	return s.objectURL(key), nil
}

func (s *uploadService) ObjectKeyFromURL(objectURL string) string {
	return utils.ParseS3ObjectKey(objectURL, s.bucketName)
}

func (s *uploadService) DeleteObject(ctx context.Context, key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}
	return s.s3Repo.Delete(ctx, s.bucketName, key)
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

	// Scale copy timeout with size (~2s/MB, min 60s, max 5m) for 20MB+ uploads
	copyTimeout := 60*time.Second + time.Duration(fileSize/(1024*1024))*2*time.Second
	if copyTimeout > 5*time.Minute {
		copyTimeout = 5 * time.Minute
	}
	copyCtx, copyCancel := context.WithTimeout(uploadCtx, copyTimeout)
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

	key := utils.GenerateS3Key(filename, s.basePath)
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
		url = s.objectURL(key)
		log.Printf("[UploadService] Upload thành công! URL: %s", url)
	}

	return &models.UploadResponse{
		URL: url,
		Key: key,
	}, nil
}
