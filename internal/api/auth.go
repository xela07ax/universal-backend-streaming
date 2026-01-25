package api

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ParseToken инкапсулирует логику валидации JWT с проверкой HMAC.
// Этот метод можно вызывать из любого места сервера.
func (s *Server) ParseToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// HMAC Validation: проверяем, что алгоритм именно HS256
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})
}

// GenerateToken создает подписанный JWT токен для пользователя.
// ttl — время жизни токена (например, 15 минут для Access или 7 дней для Refresh).
func (s *Server) GenerateToken(userID uuid.UUID, username string, role string, ttl time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"sub":   userID.String(),            // Идентификатор пользователя
		"name":  username,                   // Добавляем для фронтенда
		"role":  role,                       //  RBAC (Role-Based Access Control), где доступ определяется значением поля role.
		"admin": true,                       // Флаг прав доступа
		"exp":   time.Now().Add(ttl).Unix(), // Время истечения
		"iat":   time.Now().Unix(),          // Время выпуска
	}
	
	// Создаем токен с методом подписи HMAC HS256
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Подписываем токен нашим секретом из конфига (s.jwtSecret)
	return token.SignedString([]byte(s.jwtSecret))
}
