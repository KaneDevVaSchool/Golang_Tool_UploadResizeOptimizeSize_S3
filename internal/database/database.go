package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// DB wrap database connection và cung cấp transaction support
type DB struct {
	*sql.DB
}

// GetStats trả về thống kê connection pool
func (db *DB) GetStats() sql.DBStats {
	if db.DB == nil {
		return sql.DBStats{}
	}
	return db.DB.Stats()
}

// Tx wrap database transaction
type Tx struct {
	*sql.Tx
}

// Config cấu hình database
type Config struct {
	Driver      string
	DataSource  string
	MaxOpen     int
	MaxIdle     int
	MaxLifetime time.Duration
}

// NormalizeMySQLDSN đảm bảo DSN MySQL luôn có parseTime=true (bắt buộc để
// go-sql-driver/mysql scan cột DATETIME/TIMESTAMP thành time.Time) và
// charset=utf8mb4 (hỗ trợ đầy đủ tiếng Việt có dấu + emoji reaction).
// Không đụng vào DSN của driver khác (vd nếu cfg.Driver != "mysql").
func NormalizeMySQLDSN(dsn string) string {
	if dsn == "" {
		return dsn
	}

	base := dsn
	query := ""
	if idx := strings.Index(dsn, "?"); idx >= 0 {
		base = dsn[:idx]
		query = dsn[idx+1:]
	}

	params := map[string]string{}
	order := []string{}
	for _, pair := range strings.Split(query, "&") {
		if pair == "" {
			continue
		}
		kv := strings.SplitN(pair, "=", 2)
		key := kv[0]
		val := ""
		if len(kv) == 2 {
			val = kv[1]
		}
		if _, exists := params[key]; !exists {
			order = append(order, key)
		}
		params[key] = val
	}

	ensure := func(key, val string) {
		if _, ok := params[key]; !ok {
			params[key] = val
			order = append(order, key)
		}
	}
	ensure("parseTime", "true")
	ensure("charset", "utf8mb4")
	ensure("loc", "Local")
	// multiStatements=true cho phép 1 file migration chứa nhiều câu lệnh
	// (CREATE TABLE + CREATE INDEX + INSERT seed...) chạy trong 1 lần ExecContext,
	// giống cách cmd/migrate hiện thực thi nguyên văn nội dung mỗi file .sql.
	ensure("multiStatements", "true")

	pairs := make([]string, 0, len(order))
	for _, key := range order {
		pairs = append(pairs, key+"="+params[key])
	}

	return base + "?" + strings.Join(pairs, "&")
}

// NewDB tạo database connection mới
func NewDB(cfg Config) (*DB, error) {
	dataSource := cfg.DataSource
	if cfg.Driver == "mysql" {
		dataSource = NormalizeMySQLDSN(dataSource)
	}

	db, err := sql.Open(cfg.Driver, dataSource)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpen)
	db.SetMaxIdleConns(cfg.MaxIdle)
	db.SetConnMaxLifetime(cfg.MaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{DB: db}, nil
}

// BeginTx bắt đầu transaction mới với context
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := db.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &Tx{Tx: tx}, nil
}

// Close đóng database connection
func (db *DB) Close() error {
	return db.DB.Close()
}

// Commit commit transaction
func (tx *Tx) Commit() error {
	if err := tx.Tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// Rollback rollback transaction
func (tx *Tx) Rollback() error {
	if err := tx.Tx.Rollback(); err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}
	return nil
}
