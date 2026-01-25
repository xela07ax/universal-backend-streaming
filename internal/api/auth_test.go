package api

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGenerateAndParseToken(t *testing.T) {
	s := &Server{jwtSecret: "test-secret-2026"}

	// Подготовка тестовых данных 2026
	testID := uuid.New()
	username := "admin"
	role := "admin"
	ttl := 1 * time.Hour

	// 1. Тест успешной генерации и парсинга
	// Теперь передаем 4 аргумента: ID, name, role, ttl
	tokenString, err := s.GenerateToken(testID, username, role, ttl)
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenString)

	token, err := s.ParseToken(tokenString)
	assert.NoError(t, err)
	assert.True(t, token.Valid)

	claims, ok := token.Claims.(jwt.MapClaims)
	assert.True(t, ok)

	// Проверяем данные внутри токена
	// sub — это UUID пользователя
	assert.Equal(t, testID.String(), claims["sub"])
	// name — имя пользователя
	assert.Equal(t, username, claims["name"])
	// role — роль пользователя
	assert.Equal(t, role, claims["role"])

	// 2. Тест с неверным секретом
	wrongServer := &Server{jwtSecret: "wrong-secret"}
	_, err = wrongServer.ParseToken(tokenString)
	assert.Error(t, err, "Должна быть ошибка валидации подписи")
}
