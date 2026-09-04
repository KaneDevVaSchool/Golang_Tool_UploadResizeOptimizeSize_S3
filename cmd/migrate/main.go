package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"s3-upload-tool/internal/config"
	"s3-upload-tool/internal/database"

	_ "github.com/go-sql-driver/mysql"
)

const (
	migrationsTableName     = "schema_migrations"
	migrationsDir           = "internal/database/migrations"
	defaultMigrationTimeout = 5 * time.Minute
)

var (
	// versionRegex kiểm tra định dạng version migration (chỉ số)
	versionRegex = regexp.MustCompile(`^\d+$`)
)

type Migration struct {
	Version string
	Name    string
	Content string
}

func main() {
	var (
		upFlag      = flag.Bool("up", false, "Run all pending migrations")
		statusFlag  = flag.Bool("status", false, "Show migration status")
		versionFlag = flag.String("version", "", "Run migration to specific version")
		timeoutFlag = flag.Duration("timeout", defaultMigrationTimeout, "Timeout for migration operations")
	)
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if !cfg.Database.Enabled || cfg.Database.DataSource == "" {
		log.Fatalf("Database is not enabled or DataSource is not configured")
	}

	dbConfig := database.Config{
		Driver:      cfg.Database.Driver,
		DataSource:  cfg.Database.DataSource,
		MaxOpen:     cfg.Database.MaxOpen,
		MaxIdle:     cfg.Database.MaxIdle,
		MaxLifetime: time.Duration(cfg.Database.MaxLifetime) * time.Second,
	}

	db, err := database.NewDB(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), *timeoutFlag)
	defer cancel()

	if err := ensureMigrationsTable(ctx, db); err != nil {
		log.Fatalf("Failed to create migrations table: %v", err)
	}

	migrations, err := loadMigrations()
	if err != nil {
		log.Fatalf("Failed to load migrations: %v", err)
	}

	if len(migrations) == 0 {
		log.Println("No migrations found")
		return
	}

	appliedVersions, err := getAppliedMigrations(ctx, db)
	if err != nil {
		log.Fatalf("Failed to get applied migrations: %v", err)
	}

	if *statusFlag {
		showStatus(migrations, appliedVersions)
		return
	}

	if *upFlag {
		if err := runMigrations(ctx, db, migrations, appliedVersions, ""); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		return
	}

	if *versionFlag != "" {
		if err := runMigrations(ctx, db, migrations, appliedVersions, *versionFlag); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		return
	}

	showStatus(migrations, appliedVersions)
}

// ensureMigrationsTable tạo bảng theo dõi migrations nếu chưa tồn tại
func ensureMigrationsTable(ctx context.Context, db *database.DB) error {
	// migrationsTableName là constant cố định (không nhận input động), nên nội suy
	// trực tiếp vào DDL là an toàn - MySQL driver không có hàm quote identifier
	// public tương đương pq.QuoteIdentifier, dùng backtick thủ công cho tên bảng cố định.
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS `+"`%s`"+` (
			version VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`, migrationsTableName)

	_, err := db.ExecContext(ctx, query)
	return err
}

// loadMigrations tải tất cả file migration từ filesystem
func loadMigrations() ([]Migration, error) {
	var migrations []Migration

	// ? Tìm thư mục migrations từ nhiều vị trí có thể
	// Hỗ trợ chạy từ project root hoặc từ cmd/migrate
	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	possiblePaths := []string{
		migrationsDir,
		filepath.Join("..", "..", migrationsDir),
		filepath.Join(workDir, migrationsDir),
		filepath.Join(workDir, "..", "..", migrationsDir),
	}

	var migrationsPath string
	for _, path := range possiblePaths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			migrationsPath = path
			break
		}
	}

	if migrationsPath == "" {
		return nil, fmt.Errorf("migrations directory not found. Tried: %v", possiblePaths)
	}

	log.Printf("Loading migrations from: %s", migrationsPath)

	entries, err := os.ReadDir(migrationsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		// * Format: "001_create_uploads_table.sql" -> version "001"
		version := strings.Split(entry.Name(), "_")[0]
		if version == "" {
			log.Printf("Warning: Migration file %s has no version prefix, skipping", entry.Name())
			continue
		}

		// ! Version phải là số để sort đúng (so sánh số thay vì chuỗi)
		if !versionRegex.MatchString(version) {
			log.Printf("Warning: Invalid migration version format '%s' in %s, skipping", version, entry.Name())
			continue
		}

		if _, err := strconv.Atoi(version); err != nil {
			return nil, fmt.Errorf("invalid migration version '%s' in %s: must be numeric: %w", version, entry.Name(), err)
		}

		path := filepath.Join(migrationsPath, entry.Name())
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read migration %s: %w", entry.Name(), err)
		}

		migrations = append(migrations, Migration{
			Version: version,
			Name:    entry.Name(),
			Content: string(content),
		})
	}

	// ! Sort theo version để đảm bảo thứ tự chạy đúng
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// getAppliedMigrations trả về danh sách các migration đã apply
func getAppliedMigrations(ctx context.Context, db *database.DB) (map[string]bool, error) {
	query := fmt.Sprintf("SELECT version FROM `%s` ORDER BY version", migrationsTableName)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("failed to scan version: %w", err)
		}
		applied[version] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return applied, nil
}

// compareVersions so sánh hai version migration theo số
// Trả về: -1 nếu v1 < v2, 0 nếu v1 == v2, 1 nếu v1 > v2
func compareVersions(v1, v2 string) (int, error) {
	n1, err := strconv.Atoi(v1)
	if err != nil {
		return 0, fmt.Errorf("invalid version %s: %w", v1, err)
	}
	n2, err := strconv.Atoi(v2)
	if err != nil {
		return 0, fmt.Errorf("invalid version %s: %w", v2, err)
	}
	if n1 < n2 {
		return -1, nil
	}
	if n1 > n2 {
		return 1, nil
	}
	return 0, nil
}

// runMigrations chạy các migration pending đến version chỉ định (hoặc tất cả nếu version rỗng)
func runMigrations(ctx context.Context, db *database.DB, migrations []Migration, appliedVersions map[string]bool, targetVersion string) error {
	var pending []Migration
	for _, migration := range migrations {
		if appliedVersions[migration.Version] {
			continue
		}

		// ? So sánh số thay vì chuỗi để tránh lỗi: "10" < "2" (lexicographic) vs 10 > 2 (đúng)
		if targetVersion != "" {
			cmp, err := compareVersions(migration.Version, targetVersion)
			if err != nil {
				return fmt.Errorf("failed to compare versions: %w", err)
			}
			if cmp > 0 {
				break
			}
		}

		pending = append(pending, migration)
	}

	if len(pending) == 0 {
		log.Println("No pending migrations")
		return nil
	}

	log.Printf("Running %d pending migration(s)...", len(pending))

	for _, migration := range pending {
		log.Printf("Running migration %s: %s", migration.Version, migration.Name)

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		if _, err := tx.ExecContext(ctx, migration.Content); err != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				return fmt.Errorf("failed to execute migration %s: %w (rollback also failed: %v)", migration.Version, err, rollbackErr)
			}
			return fmt.Errorf("failed to execute migration %s: %w", migration.Version, err)
		}

		recordQuery := fmt.Sprintf(`
			INSERT INTO `+"`%s`"+` (version, name, applied_at)
			VALUES (?, ?, ?)
		`, migrationsTableName)

		if _, err := tx.ExecContext(ctx, recordQuery, migration.Version, migration.Name, time.Now()); err != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				return fmt.Errorf("failed to record migration %s: %w (rollback also failed: %v)", migration.Version, err, rollbackErr)
			}
			return fmt.Errorf("failed to record migration %s: %w", migration.Version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction for migration %s: %w", migration.Version, err)
		}

		log.Printf("✓ Migration %s applied successfully", migration.Version)
	}

	log.Println("All migrations completed successfully")
	return nil
}

// showStatus hiển thị trạng thái tất cả migrations
func showStatus(migrations []Migration, appliedVersions map[string]bool) {
	fmt.Println("\nMigration Status:")
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("%-15s %-50s %-15s\n", "Version", "Name", "Status")
	fmt.Println(strings.Repeat("-", 80))

	for _, migration := range migrations {
		status := "PENDING"
		if appliedVersions[migration.Version] {
			status = "APPLIED"
		}
		fmt.Printf("%-15s %-50s %-15s\n", migration.Version, migration.Name, status)
	}

	fmt.Println(strings.Repeat("-", 80))

	pendingCount := 0
	for _, migration := range migrations {
		if !appliedVersions[migration.Version] {
			pendingCount++
		}
	}

	fmt.Printf("\nTotal: %d migrations, %d applied, %d pending\n", len(migrations), len(migrations)-pendingCount, pendingCount)
}
