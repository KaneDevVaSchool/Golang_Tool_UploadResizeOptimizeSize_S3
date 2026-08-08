package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	defaultLogDir    = "./storage/logs"
	defaultLogPrefix = "app"
)

// SetupFromEnv configures the standard library logger to write to both
// stdout and a daily file under LOG_DIR (default ./storage/logs).
// Returns a closer that flushes/closes the log file.
func SetupFromEnv() (func(), error) {
	dir := os.Getenv("LOG_DIR")
	if dir == "" {
		dir = defaultLogDir
	}
	prefix := os.Getenv("LOG_PREFIX")
	if prefix == "" {
		prefix = defaultLogPrefix
	}
	return Setup(dir, prefix)
}

// Setup configures log output to stdout + daily rotating file.
func Setup(dir, prefix string) (func(), error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return nil, fmt.Errorf("create log dir %s: %w", absDir, err)
	}

	dw := &dailyWriter{dir: absDir, prefix: prefix}
	if err := dw.ensureFile(time.Now()); err != nil {
		return nil, err
	}

	log.SetOutput(io.MultiWriter(os.Stdout, dw))
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	today := filepath.Join(absDir, fmt.Sprintf("%s-%s.log", prefix, time.Now().Format("2006-01-02")))
	log.Printf("[Logging] Writing logs to %s", today)

	return func() {
		_ = dw.Close()
	}, nil
}

type dailyWriter struct {
	dir     string
	prefix  string
	mu      sync.Mutex
	file    *os.File
	curDate string
}

func (w *dailyWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.ensureFileLocked(time.Now()); err != nil {
		return 0, err
	}
	n, err := w.file.Write(p)
	if err == nil {
		_ = w.file.Sync()
	}
	return n, err
}

func (w *dailyWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	w.curDate = ""
	return err
}

func (w *dailyWriter) ensureFile(t time.Time) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.ensureFileLocked(t)
}

func (w *dailyWriter) ensureFileLocked(t time.Time) error {
	date := t.Format("2006-01-02")
	if w.file != nil && w.curDate == date {
		return nil
	}

	if w.file != nil {
		_ = w.file.Close()
		w.file = nil
	}

	path := filepath.Join(w.dir, fmt.Sprintf("%s-%s.log", w.prefix, date))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file %s: %w", path, err)
	}

	w.file = f
	w.curDate = date
	return nil
}
