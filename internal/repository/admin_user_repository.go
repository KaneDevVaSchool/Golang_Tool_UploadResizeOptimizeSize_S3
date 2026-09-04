package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"s3-upload-tool/internal/database"
	"s3-upload-tool/internal/models"
)

// AdminUserRepository quản lý CRUD cho admin_users - tài khoản quản trị
// đăng nhập qua Google OAuth, không có mật khẩu nội bộ.
type AdminUserRepository interface {
	FindByGoogleSub(ctx context.Context, googleSub string) (*models.AdminUser, error)
	FindByEmail(ctx context.Context, email string) (*models.AdminUser, error)
	Create(ctx context.Context, user *models.AdminUser) (*models.AdminUser, error)
	UpdateLastLogin(ctx context.Context, id int64) error
}

type adminUserRepository struct {
	db *database.DB
}

func NewAdminUserRepository(db *database.DB) AdminUserRepository {
	return &adminUserRepository{db: db}
}

const adminUserSelectColumns = `id, google_sub, email, name, avatar_url, role, is_active, last_login_at, created_at, updated_at`

func scanAdminUser(scanner interface {
	Scan(dest ...any) error
}) (*models.AdminUser, error) {
	u := &models.AdminUser{}
	var name, avatarURL sql.NullString
	var lastLoginAt sql.NullTime

	err := scanner.Scan(
		&u.ID, &u.GoogleSub, &u.Email, &name, &avatarURL, &u.Role, &u.IsActive,
		&lastLoginAt, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	u.Name = name.String
	u.AvatarURL = avatarURL.String
	if lastLoginAt.Valid {
		t := lastLoginAt.Time
		u.LastLoginAt = &t
	}
	return u, nil
}

func (r *adminUserRepository) FindByGoogleSub(ctx context.Context, googleSub string) (*models.AdminUser, error) {
	query := fmt.Sprintf(`SELECT %s FROM admin_users WHERE google_sub = ?`, adminUserSelectColumns)
	row := r.db.QueryRowContext(ctx, query, googleSub)
	user, err := scanAdminUser(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find admin user by google_sub: %w", err)
	}
	return user, nil
}

func (r *adminUserRepository) FindByEmail(ctx context.Context, email string) (*models.AdminUser, error) {
	query := fmt.Sprintf(`SELECT %s FROM admin_users WHERE email = ?`, adminUserSelectColumns)
	row := r.db.QueryRowContext(ctx, query, email)
	user, err := scanAdminUser(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find admin user by email: %w", err)
	}
	return user, nil
}

func (r *adminUserRepository) Create(ctx context.Context, user *models.AdminUser) (*models.AdminUser, error) {
	now := time.Now()
	query := `
		INSERT INTO admin_users (google_sub, email, name, avatar_url, role, is_active, last_login_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	role := user.Role
	if role == "" {
		role = models.AdminRoleAdmin
	}

	result, err := r.db.ExecContext(ctx, query,
		user.GoogleSub, user.Email, user.Name, user.AvatarURL, role, true, now, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create admin user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}

	user.ID = id
	user.Role = role
	user.IsActive = true
	user.LastLoginAt = &now
	user.CreatedAt = now
	user.UpdatedAt = now
	return user, nil
}

func (r *adminUserRepository) UpdateLastLogin(ctx context.Context, id int64) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `UPDATE admin_users SET last_login_at = ?, updated_at = ? WHERE id = ?`, now, now, id)
	if err != nil {
		return fmt.Errorf("failed to update last_login_at: %w", err)
	}
	return nil
}
