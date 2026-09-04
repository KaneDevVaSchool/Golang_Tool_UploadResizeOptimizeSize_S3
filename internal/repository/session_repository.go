package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"s3-upload-tool/internal/database"
	"s3-upload-tool/internal/models"
)

// SessionRepository quản lý admin_sessions - session lưu DB (không
// in-memory) để sống sót qua restart server.
type SessionRepository interface {
	Create(ctx context.Context, session *models.AdminSession) error
	// FindByToken trả về session kèm admin user tương ứng (join sẵn), hoặc
	// (nil, nil, nil) nếu không tìm thấy - handler tự quyết định 401.
	FindByToken(ctx context.Context, token string) (*models.AdminSession, *models.AdminUser, error)
	Delete(ctx context.Context, token string) error
	DeleteExpired(ctx context.Context) (int64, error)
}

type sessionRepository struct {
	db *database.DB
}

func NewSessionRepository(db *database.DB) SessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) Create(ctx context.Context, session *models.AdminSession) error {
	query := `INSERT INTO admin_sessions (id, admin_user_id, expires_at, created_at) VALUES (?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, session.ID, session.AdminUserID, session.ExpiresAt, session.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

func (r *sessionRepository) FindByToken(ctx context.Context, token string) (*models.AdminSession, *models.AdminUser, error) {
	query := `
		SELECT
			s.id, s.admin_user_id, s.expires_at, s.created_at,
			u.id, u.google_sub, u.email, u.name, u.avatar_url, u.role, u.is_active, u.last_login_at, u.created_at, u.updated_at
		FROM admin_sessions s
		JOIN admin_users u ON u.id = s.admin_user_id
		WHERE s.id = ?
	`
	row := r.db.QueryRowContext(ctx, query, token)

	session := &models.AdminSession{}
	user := &models.AdminUser{}
	var name, avatarURL sql.NullString
	var lastLoginAt sql.NullTime

	err := row.Scan(
		&session.ID, &session.AdminUserID, &session.ExpiresAt, &session.CreatedAt,
		&user.ID, &user.GoogleSub, &user.Email, &name, &avatarURL, &user.Role, &user.IsActive, &lastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find session: %w", err)
	}

	user.Name = name.String
	user.AvatarURL = avatarURL.String
	if lastLoginAt.Valid {
		t := lastLoginAt.Time
		user.LastLoginAt = &t
	}

	return session, user, nil
}

func (r *sessionRepository) Delete(ctx context.Context, token string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM admin_sessions WHERE id = ?`, token)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

func (r *sessionRepository) DeleteExpired(ctx context.Context) (int64, error) {
	result, err := r.db.ExecContext(ctx, `DELETE FROM admin_sessions WHERE expires_at < ?`, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired sessions: %w", err)
	}
	return result.RowsAffected()
}
