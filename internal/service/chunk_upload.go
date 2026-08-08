package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"s3-upload-tool/internal/models"
	"s3-upload-tool/internal/repository"
	"s3-upload-tool/internal/utils"

	"github.com/google/uuid"
)

const (
	chunkSessionTTL    = 45 * time.Minute
	chunkCleanupEvery  = 5 * time.Minute
	maxChunkSessions   = 64
	maxChunksPerUpload = 64 // absoluteMax/chunkSize safety (e.g. 200/20 = 10)
)

var (
	ErrChunkSessionNotFound = errors.New("upload session not found or expired")
	ErrChunkSessionBusy     = errors.New("upload session is already completing")
	ErrTooManySessions      = errors.New("too many concurrent chunk upload sessions")
)

type ChunkSession struct {
	ID          string
	Filename    string
	TotalSize   int64
	ChunkSize   int64
	TotalChunks int
	Dir         string
	Received    map[int]bool
	CreatedAt   time.Time
	Completing  bool
	mu          sync.Mutex
}

type ChunkUploadService interface {
	Init(filename string, totalSize int64) (*ChunkSession, error)
	SaveChunk(uploadID string, index int, reader io.Reader, declaredSize int64) error
	Complete(ctx context.Context, uploadID string) (resp *models.UploadResponse, size int64, filename string, err error)
	Abort(uploadID string)
	Stop()
}

type chunkUploadService struct {
	s3Repo             repository.S3Repository
	bucketName         string
	basePath           string
	region             string
	uploadDir          string
	chunkSize          int64
	absoluteMax        int64
	uploadTimeout      time.Duration
	useACL             bool
	usePresignedURL    bool
	presignedURLExpiry int
	endpoint           string
	forcePathStyle     bool

	mu       sync.Mutex
	sessions map[string]*ChunkSession
	stopCh   chan struct{}
}

func NewChunkUploadService(
	s3Repo repository.S3Repository,
	bucketName, region, uploadDir string,
	chunkSize, absoluteMax int64,
	uploadTimeout time.Duration,
	useACL, usePresignedURL bool,
	presignedURLExpiry int,
	endpoint string,
	forcePathStyle bool,
	basePath string,
) ChunkUploadService {
	if chunkSize <= 0 {
		chunkSize = 20 << 20
	}
	if absoluteMax < chunkSize {
		absoluteMax = chunkSize
	}
	s := &chunkUploadService{
		s3Repo:             s3Repo,
		bucketName:         bucketName,
		basePath:           basePath,
		region:             region,
		uploadDir:          uploadDir,
		chunkSize:          chunkSize,
		absoluteMax:        absoluteMax,
		uploadTimeout:      uploadTimeout,
		useACL:             useACL,
		usePresignedURL:    usePresignedURL,
		presignedURLExpiry: presignedURLExpiry,
		endpoint:           endpoint,
		forcePathStyle:     forcePathStyle,
		sessions:           make(map[string]*ChunkSession),
		stopCh:             make(chan struct{}),
	}
	go s.cleanupLoop()
	return s
}

func parseUploadID(uploadID string) (string, error) {
	id, err := uuid.Parse(uploadID)
	if err != nil {
		return "", ErrChunkSessionNotFound
	}
	return id.String(), nil
}

func (s *chunkUploadService) Init(filename string, totalSize int64) (*ChunkSession, error) {
	sanitized, err := utils.SanitizeFilename(filename)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidFileFormat, err)
	}
	if !utils.IsAllowedFileType(sanitized) {
		return nil, ErrInvalidFileFormat
	}
	if totalSize <= 0 {
		return nil, fmt.Errorf("invalid total size")
	}
	if totalSize > s.absoluteMax {
		return nil, NewFileSizeError(totalSize, s.absoluteMax,
			fmt.Sprintf("file quá lớn: %s (giới hạn tuyệt đối: %s)",
				utils.FormatFileSize(totalSize), utils.FormatFileSize(s.absoluteMax)))
	}

	totalChunks := int((totalSize + s.chunkSize - 1) / s.chunkSize)
	if totalChunks < 1 || totalChunks > maxChunksPerUpload {
		return nil, fmt.Errorf("invalid chunk count for file size")
	}

	s.mu.Lock()
	if len(s.sessions) >= maxChunkSessions {
		s.mu.Unlock()
		return nil, ErrTooManySessions
	}
	s.mu.Unlock()

	id := uuid.NewString()
	dir := filepath.Join(s.uploadDir, "chunks", id)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create chunk dir: %w", err)
	}

	session := &ChunkSession{
		ID:          id,
		Filename:    sanitized,
		TotalSize:   totalSize,
		ChunkSize:   s.chunkSize,
		TotalChunks: totalChunks,
		Dir:         dir,
		Received:    make(map[int]bool, totalChunks),
		CreatedAt:   time.Now(),
	}

	s.mu.Lock()
	if len(s.sessions) >= maxChunkSessions {
		s.mu.Unlock()
		_ = os.RemoveAll(dir)
		return nil, ErrTooManySessions
	}
	s.sessions[id] = session
	s.mu.Unlock()

	log.Printf("[ChunkUpload] Init session=%s file=%s size=%s chunks=%d",
		id, sanitized, utils.FormatFileSize(totalSize), totalChunks)
	return session, nil
}

func (s *chunkUploadService) getSession(uploadID string) (*ChunkSession, error) {
	id, err := parseUploadID(uploadID)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[id]
	if !ok {
		return nil, ErrChunkSessionNotFound
	}
	if time.Since(session.CreatedAt) > chunkSessionTTL {
		delete(s.sessions, id)
		dir := session.Dir
		go func() { _ = os.RemoveAll(dir) }()
		return nil, ErrChunkSessionNotFound
	}
	return session, nil
}

func (s *chunkUploadService) SaveChunk(uploadID string, index int, reader io.Reader, declaredSize int64) error {
	session, err := s.getSession(uploadID)
	if err != nil {
		return err
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if session.Completing {
		return ErrChunkSessionBusy
	}
	if index < 0 || index >= session.TotalChunks {
		return fmt.Errorf("invalid chunk index %d", index)
	}

	expected := session.ChunkSize
	if index == session.TotalChunks-1 {
		expected = session.TotalSize - int64(index)*session.ChunkSize
	}
	// Multipart FileHeader.Size can be 0 on some clients — only enforce when declared.
	if declaredSize > 0 && declaredSize != expected {
		return fmt.Errorf("chunk size mismatch: got %d, expected %d", declaredSize, expected)
	}
	if declaredSize > session.ChunkSize {
		return fmt.Errorf("chunk exceeds max chunk size %s", utils.FormatFileSize(session.ChunkSize))
	}

	partPath := filepath.Join(session.Dir, fmt.Sprintf("part_%06d", index))
	tmpPath := partPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create chunk file: %w", err)
	}

	written, copyErr := io.Copy(f, io.LimitReader(reader, expected+1))
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to write chunk: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to close chunk file: %w", closeErr)
	}
	if written != expected {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("incomplete chunk: wrote %d of %d bytes", written, expected)
	}
	if err := os.Rename(tmpPath, partPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to finalize chunk: %w", err)
	}

	session.Received[index] = true
	log.Printf("[ChunkUpload] session=%s chunk=%d/%d saved (%s)",
		uploadID, index+1, session.TotalChunks, utils.FormatFileSize(written))
	return nil
}

func (s *chunkUploadService) Complete(ctx context.Context, uploadID string) (*models.UploadResponse, int64, string, error) {
	session, err := s.getSession(uploadID)
	if err != nil {
		return nil, 0, "", err
	}

	session.mu.Lock()
	if session.Completing {
		session.mu.Unlock()
		return nil, 0, "", ErrChunkSessionBusy
	}
	for i := 0; i < session.TotalChunks; i++ {
		if !session.Received[i] {
			session.mu.Unlock()
			return nil, 0, "", fmt.Errorf("missing chunk %d", i)
		}
	}
	session.Completing = true
	filename := session.Filename
	totalSize := session.TotalSize
	totalChunks := session.TotalChunks
	dir := session.Dir
	session.mu.Unlock()

	// Ensure failed completes can retry
	success := false
	defer func() {
		if !success {
			session.mu.Lock()
			session.Completing = false
			session.mu.Unlock()
		}
	}()

	assembled, err := os.CreateTemp(s.uploadDir, "assembled-*"+filepath.Ext(filename))
	if err != nil {
		return nil, 0, "", fmt.Errorf("failed to create assembled file: %w", err)
	}
	assembledPath := assembled.Name()
	defer func() {
		assembled.Close()
		_ = os.Remove(assembledPath)
	}()

	var totalWritten int64
	for i := 0; i < totalChunks; i++ {
		partPath := filepath.Join(dir, fmt.Sprintf("part_%06d", i))
		part, err := os.Open(partPath)
		if err != nil {
			return nil, 0, "", fmt.Errorf("failed to open chunk %d: %w", i, err)
		}
		n, copyErr := io.Copy(assembled, part)
		part.Close()
		if copyErr != nil {
			return nil, 0, "", fmt.Errorf("failed to assemble chunk %d: %w", i, copyErr)
		}
		totalWritten += n
	}

	if totalWritten != totalSize {
		return nil, 0, "", fmt.Errorf("assembled size mismatch: got %d, expected %d", totalWritten, totalSize)
	}

	if _, err := assembled.Seek(0, 0); err != nil {
		return nil, 0, "", fmt.Errorf("failed to seek assembled file: %w", err)
	}

	if err := utils.ValidateFileContent(assembled, filename); err != nil {
		return nil, 0, "", fmt.Errorf("%w: %v", ErrInvalidFileFormat, err)
	}
	if _, err := assembled.Seek(0, 0); err != nil {
		return nil, 0, "", err
	}

	uploadCtx := ctx
	if s.uploadTimeout > 0 {
		var cancel context.CancelFunc
		uploadCtx, cancel = context.WithTimeout(ctx, s.uploadTimeout)
		defer cancel()
	}

	key := utils.GenerateS3Key(filename, s.basePath)
	contentType := utils.GetContentType(filename)
	if _, err := s.s3Repo.Upload(uploadCtx, s.bucketName, key, assembled, contentType, s.useACL); err != nil {
		return nil, 0, "", fmt.Errorf("%w: %v", ErrUploadToS3, err)
	}

	var url string
	if s.usePresignedURL {
		expiry := time.Duration(s.presignedURLExpiry) * time.Minute
		if expiry == 0 {
			expiry = 60 * time.Minute
		}
		presignedURL, err := s.s3Repo.GeneratePresignedURL(uploadCtx, s.bucketName, key, expiry)
		if err != nil {
			return nil, 0, "", fmt.Errorf("%w: %v", ErrUploadToS3, err)
		}
		url = presignedURL
	} else {
		url = utils.BuildS3ObjectURL(s.bucketName, s.region, key, s.endpoint, s.forcePathStyle)
	}

	success = true
	s.removeSession(uploadID)
	log.Printf("[ChunkUpload] Complete session=%s key=%s size=%s", uploadID, key, utils.FormatFileSize(totalWritten))

	return &models.UploadResponse{URL: url, Key: key}, totalWritten, filename, nil
}

func (s *chunkUploadService) Abort(uploadID string) {
	s.removeSession(uploadID)
}

func (s *chunkUploadService) removeSession(uploadID string) {
	id, err := parseUploadID(uploadID)
	if err != nil {
		return
	}
	s.mu.Lock()
	session, ok := s.sessions[id]
	if ok {
		delete(s.sessions, id)
	}
	s.mu.Unlock()
	if ok {
		_ = os.RemoveAll(session.Dir)
	}
}

func (s *chunkUploadService) cleanupLoop() {
	ticker := time.NewTicker(chunkCleanupEvery)
	defer ticker.Stop()
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			var staleDirs []string
			s.mu.Lock()
			now := time.Now()
			for id, session := range s.sessions {
				session.mu.Lock()
				busy := session.Completing
				expired := now.Sub(session.CreatedAt) > chunkSessionTTL
				session.mu.Unlock()
				if expired && !busy {
					staleDirs = append(staleDirs, session.Dir)
					delete(s.sessions, id)
					log.Printf("[ChunkUpload] Expired session cleaned: %s", id)
				}
			}
			s.mu.Unlock()
			for _, dir := range staleDirs {
				_ = os.RemoveAll(dir)
			}
		}
	}
}

func (s *chunkUploadService) Stop() {
	select {
	case <-s.stopCh:
	default:
		close(s.stopCh)
	}
	s.mu.Lock()
	dirs := make([]string, 0, len(s.sessions))
	for id, session := range s.sessions {
		dirs = append(dirs, session.Dir)
		delete(s.sessions, id)
	}
	s.mu.Unlock()
	for _, dir := range dirs {
		_ = os.RemoveAll(dir)
	}
}
