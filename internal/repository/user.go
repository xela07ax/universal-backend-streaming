package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	RoleAdmin     = "admin"
	RoleModerator = "moderator"
	RoleUser      = "user"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Role         string    `db:"role"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // Никогда не отдаем хеш в JSON
}

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// GetByUsername ищет пользователя для проверки пароля
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*User, error) {
	var u User
	// Добавляем role в выборку
	query := `
		SELECT id, username, password_hash, role 
		FROM users 
		WHERE username = $1
	`

	err := r.db.QueryRow(ctx, query, username).Scan(
		&u.ID,
		&u.Username,
		&u.PasswordHash,
		&u.Role, // Сканируем роль в структуру
	)

	if err != nil {
		// В 2026 году важно отличать "не нашел" от "ошибка связи"
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("user %s not found", username)
		}
		return nil, fmt.Errorf("repository: failed to fetch user: %w", err)
	}

	return &u, nil
}
