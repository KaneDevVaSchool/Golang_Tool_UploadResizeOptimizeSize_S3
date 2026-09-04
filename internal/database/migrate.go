package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const migrationsTableName = "schema_migrations"

var migrationVersionRegex = regexp.MustCompile(`^\d+$`)

type migrationFile struct {
	Version string
	Name    string
	Content string
}

// RunMigrations đọc mọi file .sql trong dir theo thứ tự version (prefix số,
// vd "001_..."), chạy từng file trong 1 transaction, và ghi version đã áp
// dụng vào bảng schema_migrations để idempotent - gọi lại nhiều lần an toàn,
// chỉ các file mới sẽ được chạy.
//
// Đây là cơ chế tự động dùng lúc container khởi động (bật/tắt qua
// DATABASE_AUTO_MIGRATE). cmd/migrate vẫn là công cụ CLI riêng để chạy
// migration thủ công / xem status, dùng chung format bảng/cột nhưng độc lập
// vòng đời với container.
func RunMigrations(ctx context.Context, db *DB, dir string) error {
	if err := ensureMigrationsTable(ctx, db); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	files, err := loadMigrationFiles(dir)
	if err != nil {
		return fmt.Errorf("failed to load migration files: %w", err)
	}
	if len(files) == 0 {
		log.Printf("[Migrate] Không tìm thấy migration nào trong %s", dir)
		return nil
	}

	applied, err := appliedMigrationVersions(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to read applied migrations: %w", err)
	}

	pending := 0
	for _, f := range files {
		if applied[f.Version] {
			continue
		}
		pending++

		if err := applyMigration(ctx, db, f); err != nil {
			return fmt.Errorf("migration %s (%s) failed: %w", f.Version, f.Name, err)
		}
		log.Printf("[Migrate] ✓ Applied %s (%s)", f.Version, f.Name)
	}

	if pending == 0 {
		log.Println("[Migrate] Không có migration mới cần chạy")
	} else {
		log.Printf("[Migrate] Đã áp dụng %d migration mới", pending)
	}
	return nil
}

func ensureMigrationsTable(ctx context.Context, db *DB) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS `+"`%s`"+` (
			version VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`, migrationsTableName)
	_, err := db.ExecContext(ctx, query)
	return err
}

func loadMigrationFiles(dir string) ([]migrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []migrationFile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		version := strings.SplitN(entry.Name(), "_", 2)[0]
		if !migrationVersionRegex.MatchString(version) {
			log.Printf("[Migrate] Bỏ qua file không đúng định dạng version: %s", entry.Name())
			continue
		}

		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", entry.Name(), err)
		}

		files = append(files, migrationFile{
			Version: version,
			Name:    entry.Name(),
			Content: string(content),
		})
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Version < files[j].Version })
	return files, nil
}

func appliedMigrationVersions(ctx context.Context, db *DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("SELECT version FROM `%s`", migrationsTableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}
	return applied, rows.Err()
}

func applyMigration(ctx context.Context, db *DB, f migrationFile) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	if _, err := tx.ExecContext(ctx, f.Content); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("exec failed: %w (rollback also failed: %v)", err, rbErr)
		}
		return fmt.Errorf("exec failed: %w", err)
	}

	recordQuery := fmt.Sprintf("INSERT INTO `%s` (version, name, applied_at) VALUES (?, ?, ?)", migrationsTableName)
	if _, err := tx.ExecContext(ctx, recordQuery, f.Version, f.Name, time.Now()); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("failed to record migration: %w (rollback also failed: %v)", err, rbErr)
		}
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit()
}
