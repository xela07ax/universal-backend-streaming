package types

import (
	"context"

	"github.com/google/uuid"
)

// ContextKey — общий тип для всех ключей контекста в Hydro Engine
type ContextKey string

const (
	UserIDKey   ContextKey = "user_id"
	UserRoleKey ContextKey = "user_role"
)

// GetUserID Публичная функция для извлечения ID из любого контекста (защита от коллизий). Если написать context.WithValue(ctx, "user_id", userID),
// то любая библиотека, которую вы подключите в будущем, может сделать так же. Это приведет к трудноотловимым багам.
// Собственный тип contextKey гарантирует, что только ваш код сможет обратиться к этому значению.
func GetUserID(ctx context.Context) (uuid.UUID, bool) {
	uid, ok := ctx.Value(UserIDKey).(uuid.UUID)
	return uid, ok
}

func GetUserRole(ctx context.Context) string {
	role := ctx.Value(UserRoleKey).(string)
	return role
}
