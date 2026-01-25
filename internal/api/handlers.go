package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/xela07ax/universal-backend-streaming/internal/repository"
	"github.com/xela07ax/universal-backend-streaming/internal/types"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func (s *Server) handleGetVideoURL(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	asset, err := s.media.GetAssetByID(r.Context(), id)
	if err != nil {
		s.respondError(w, http.StatusNotFound, "Video not found")
		return
	}

	streamingURL := s.video.BuildURL(asset.StoragePath)

	// ВАЖНО: структура ответа должна совпадать с тем, что ищет фронтенд
	s.respond(w, http.StatusOK, map[string]string{
		"url":   streamingURL,
		"title": asset.Title,
	})
}

// handleAdminUploadAsset принимает видеофайл и метаданные
func (s *Server) handleAdminUploadAsset(w http.ResponseWriter, r *http.Request) {
	// 1. Извлекаем данные из контекста (ID и Роль)
	userID, ok := types.GetUserID(r.Context())
	if !ok {
		s.respondError(w, http.StatusUnauthorized, "Не удалось идентифицировать пользователя")
		return
	}
	role := types.GetUserRole(r.Context())
	// ПРОВЕРКА РОЛИ: только admin может продолжать
	if role != "admin" {
		s.logger.Warn("🚫 Unauthorized upload attempt",
			zap.String("user_id", userID.String()),
			zap.String("role", role), // Теперь тут будет "user" или пусто, но не null
		)
		s.respondError(w, http.StatusForbidden, "У вас нет прав для загрузки видео")
		return
	}

	// 2. Лимит на чтение (505MB)
	r.Body = http.MaxBytesReader(w, r.Body, 505<<20)

	// 3. Парсим форму
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		s.logger.Error("Upload: parse form error", zap.Error(err))
		s.respondError(w, http.StatusRequestEntityTooLarge, "Файл слишком большой")
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "Поле 'video' не найдено")
		return
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			s.logger.Warn("Failed to close uploaded multipart file", zap.Error(closeErr))
		}
	}()

	// 4. Подготовка путей
	title := r.FormValue("title")
	if title == "" {
		title = header.Filename
	}

	ext := filepath.Ext(header.Filename)
	fileName := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	uploadDir := filepath.Join("web", "dist", "uploads")

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		s.logger.Error("Upload: mkdir error", zap.Error(err))
		s.respondError(w, http.StatusInternalServerError, "Ошибка хранилища")
		return
	}

	storagePath := filepath.Join("uploads", fileName)
	fullPath := filepath.Join("web", "dist", storagePath)

	// 5. Сохранение файла с механизмом отката
	dst, err := os.Create(fullPath)
	if err != nil {
		s.logger.Error("Upload: create file error", zap.Error(err))
		s.respondError(w, http.StatusInternalServerError, "Ошибка создания файла")
		return
	}

	var success bool
	defer func() {
		// 1. Закрываем файл и проверяем ошибку
		if err := dst.Close(); err != nil {
			s.logger.Error("❌ Upload: failed to close destination file", zap.Error(err))
			// Если закрытие не удалось, мы не можем считать операцию успешной
			success = false
		}

		// 2. Если в процессе возникла ошибка или закрытие файла упало — удаляем мусор
		if !success {
			s.logger.Warn("Rolling back: deleting file", zap.String("path", fullPath))
			if err := os.Remove(fullPath); err != nil {
				// Требует внимания администратора.
				s.logger.Warn("⚠️ Failed to remove orphaned file",
					zap.String("path", fullPath),
					zap.Error(err),
				)
			}
		}
	}()

	if _, err := io.Copy(dst, file); err != nil {
		s.logger.Error("Upload: copy error", zap.Error(err))
		s.respondError(w, http.StatusInternalServerError, "Ошибка записи")
		return
	}

	// Явно закрываем файл, чтобы освободить дескриптор для ОС
	if err := dst.Close(); err != nil {
		s.logger.Error("❌ Upload: failed to close file", zap.Error(err))
		s.respondError(w, http.StatusInternalServerError, "Ошибка при сохранении файла")
		return
	}

	// 6. Запись в БД
	asset := &repository.MediaAsset{
		ID:          uuid.New(),
		OwnerID:     userID, // Используем динамический ID из токена
		Title:       title,
		StoragePath: filepath.ToSlash(storagePath),
		Status:      "ready",
		Metadata: map[string]interface{}{
			"size": header.Size,
			"type": header.Header.Get("Content-Type"),
		},
	}

	if err := s.media.SaveAsset(r.Context(), asset); err != nil {
		s.logger.Error("Upload: DB save error", zap.Error(err))
		s.respondError(w, http.StatusInternalServerError, "Ошибка записи в базу")
		return
	}

	success = true // Флаг для defer: файл удалять не нужно
	s.logger.Info("Video uploaded successfully", zap.String("user_id", userID.String()))
	s.respond(w, http.StatusCreated, asset)
}

// handleHealth проверяет работоспособность сервера и критических зависимостей (БД).
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// 1. Проверяем соединение с PostgreSQL
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	err := s.db.Ping(ctx)
	if err != nil {
		s.logger.Error("Healthcheck failed: database unreachable", zap.Error(err))
		s.respondError(w, http.StatusServiceUnavailable, "Database connection lost")
		return
	}

	// 2. Если всё хорошо
	s.respond(w, http.StatusOK, map[string]string{
		"status": "healthy",
		"db":     "connected",
	})
}

// LoginRequest описывает входящие данные от Vue-фронтенда
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// handleLogin — проверяет учетные данные и выдает JWT
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// 1. Получаем пользователя из БД
	user, err := s.users.GetByUsername(r.Context(), req.Username)
	if err != nil {
		s.logger.Warn("Login failed: user not found", zap.String("user", req.Username))
		s.respondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// 2. Сверяем пароль
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		s.logger.Warn("Login failed: wrong password", zap.String("user", req.Username))
		s.respondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// 3. Подготовка TTL (сначала определяем, потом используем)
	accessTTL := viper.GetDuration("auth.access_token_ttl")
	if accessTTL == 0 {
		accessTTL = 15 * time.Minute
	}
	refreshTTL := viper.GetDuration("auth.refresh_token_ttl")
	if refreshTTL == 0 {
		refreshTTL = 168 * time.Hour
	}

	// 4. Генерируем токены с полным набором данных
	// Теперь передаем: ID, Username, Role и TTL
	accessToken, err := s.GenerateToken(user.ID, user.Username, user.Role, accessTTL)
	if err != nil {
		s.logger.Error("Token access generation failed", zap.Error(err))
		s.respondError(w, http.StatusInternalServerError, "Internal error")
		return
	}

	refreshToken, err := s.GenerateToken(user.ID, user.Username, user.Role, refreshTTL)
	if err != nil {
		s.logger.Error("Token refresh generation failed", zap.Error(err))
		s.respondError(w, http.StatusInternalServerError, "Internal error")
		return
	}

	// 5. Сохраняем сессию в Redis (связываем рефреш-токен с ID пользователя)
	ctx := r.Context()
	err = s.rdb.Set(ctx, "session:"+refreshToken, user.ID.String(), refreshTTL).Err()
	if err != nil {
		s.logger.Error("Redis save error", zap.Error(err))
		s.respondError(w, http.StatusInternalServerError, "Failed to save session")
		return
	}

	// 6. Устанавливаем Refresh Token в HttpOnly куку
	http.SetCookie(w, &http.Cookie{
		Name:     "hydro_refresh_token",
		Value:    refreshToken,
		Path:     "/api/v1/refresh",
		HttpOnly: true,
		Secure:   true, // Обязательно для 2026 года
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(refreshTTL.Seconds()),
	})

	// Лог для продакшена (минимум данных)
	s.logger.Info("User logged in", zap.String("role", user.Role))

	// Детальный лог для разработки/отладки
	s.logger.Debug("Login details",
		zap.String("user", user.Username),
		zap.String("id", user.ID.String()),
		zap.String("role", user.Role),
	)

	// 7. Возвращаем ответ фронтенду
	s.respond(w, http.StatusOK, map[string]interface{}{
		"token": accessToken,
		"user": map[string]string{
			"username": user.Username,
			"role":     user.Role,
		},
	})
}

// handleRefresh проверяет Refresh-токен в Redis и выдает новую пару токенов.
func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	// 1. Извлекаем Refresh-токен из защищенной куки
	cookie, err := r.Cookie("hydro_refresh_token")
	if err != nil {
		s.respondError(w, http.StatusUnauthorized, "Refresh token missing")
		return
	}
	refreshToken := cookie.Value

	// 2. Валидация токена
	token, err := s.ParseToken(refreshToken)
	if err != nil || !token.Valid {
		s.logger.Warn("Refresh failed: invalid token signature", zap.Error(err))
		s.respondError(w, http.StatusUnauthorized, "Invalid refresh token")
		return
	}

	// 3. Извлекаем данные из Claims (нам нужны ID, Username и Role)
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		s.respondError(w, http.StatusUnauthorized, "Invalid token claims")
		return
	}

	userIDStr, _ := claims["sub"].(string)
	username, _ := claims["name"].(string)
	role, _ := claims["role"].(string)
	userID, _ := uuid.Parse(userIDStr)

	// 4. ПРОВЕРКА В REDIS: Существует ли эта сессия?
	ctx := r.Context()
	// В Redis мы храним связь токен -> userID
	storedID, err := s.rdb.Get(ctx, "session:"+refreshToken).Result()
	if err != nil || storedID != userIDStr {
		s.logger.Warn("Refresh failed: session revoked or mismatch", zap.String("userID", userIDStr))
		s.respondError(w, http.StatusUnauthorized, "Session expired or revoked")
		return
	}

	// 5. Подготовка TTL
	accessTTL := viper.GetDuration("auth.access_token_ttl")
	if accessTTL == 0 {
		accessTTL = 15 * time.Minute
	}
	refreshTTL := viper.GetDuration("auth.refresh_token_ttl")
	if refreshTTL == 0 {
		refreshTTL = 168 * time.Hour
	}

	// 6. ГЕНЕРАЦИЯ НОВОЙ ПАРЫ (с актуальными данными)
	newAccessToken, err := s.GenerateToken(userID, username, role, accessTTL)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Internal error")
		return
	}
	newRefreshToken, err := s.GenerateToken(userID, username, role, refreshTTL)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Internal error")
		return
	}

	// 7. РОТАЦИЯ В REDIS (Удаляем старый, пишем новый)
	s.rdb.Del(ctx, "session:"+refreshToken)
	err = s.rdb.Set(ctx, "session:"+newRefreshToken, userIDStr, refreshTTL).Err()
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to rotate session")
		return
	}

	// 8. ОБНОВЛЯЕМ КУКУ
	http.SetCookie(w, &http.Cookie{
		Name:     "hydro_refresh_token",
		Value:    newRefreshToken,
		Path:     "/api/v1/refresh",
		HttpOnly: true,
		Secure:   viper.GetBool("auth.secure_cookie"),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(refreshTTL.Seconds()),
	})

	s.logger.Debug("Token rotated", zap.String("id", userIDStr))
	s.respond(w, http.StatusOK, map[string]string{
		"token": newAccessToken,
	})
}

// handleLogout — Подтверждает выход.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	// 1. Пытаемся достать Refresh-токен из куки
	cookie, err := r.Cookie("hydro_refresh_token")
	if err == nil {
		refreshToken := cookie.Value
		// 2. УДАЛЯЕМ ИЗ REDIS: Теперь этот токен больше никогда не сработает
		ctx := r.Context()
		s.rdb.Del(ctx, "session:"+refreshToken)

		s.logger.Info("Session revoked in Redis", zap.String("token_tail", refreshToken[len(refreshToken)-8:]))
	}

	// 3. ОБНУЛЯЕМ КУКУ В БРАУЗЕРЕ (ставим MaxAge: -1)
	http.SetCookie(w, &http.Cookie{
		Name:     "hydro_refresh_token",
		Value:    "",
		Path:     "/api/v1/refresh",
		HttpOnly: true,
		MaxAge:   -1, // Приказывает браузеру немедленно удалить куку
	})

	s.respond(w, http.StatusOK, map[string]string{
		"message": "Successfully logged out and session revoked",
	})
}
