package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
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

// NewDB tạo database connection mới
func NewDB(cfg Config) (*DB, error) {
	db, err := sql.Open(cfg.Driver, cfg.DataSource)
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
