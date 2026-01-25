package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/xela07ax/universal-backend-streaming/internal/types"
	"go.uber.org/zap"
)

// AuthMiddleware middleware для защиты админских эндпоинтов
func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Извлекаем заголовок
		tokenHeader := r.Header.Get("Authorization")
		if tokenHeader == "" {
			s.respondError(w, http.StatusUnauthorized, "Токен авторизации отсутствует")
			return
		}

		// 2. Максимально быстрая очистка префикса
		tokenString := strings.TrimPrefix(tokenHeader, "Bearer ")

		// 2. Парсинг и валидация
		token, err := s.ParseToken(tokenString)
		if err != nil {
			// Логируем реальную причину ошибки (expired, bad signature и т.д.)
			// 1. Детальная инфа (ошибка, кусок токена) только для разработчика в Debug
			s.logger.Debug("🔒 JWT Validation Details",
				zap.Error(err),
				zap.String("token_snippet", tokenString[:10]+"..."),
			)

			// 2. В Warn пишем только факт, если это действительно важно (например, неверная подпись)
			// Если токен просто истек (Expired), это обычно не логируют.
			if !strings.Contains(err.Error(), "expired") {
				s.logger.Warn("⚠️  Unauthorized access attempt", zap.String("remote_addr", r.RemoteAddr))
			}

			s.respondError(w, http.StatusUnauthorized, "Невалидный токен")
			return
		}

		// 3. Извлекаем Claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			s.respondError(w, http.StatusUnauthorized, "Ошибка структуры токена")
			return
		}

		// 4. Проверка прав администратора
		// Извлекаем роль из claims
		role, ok := claims["role"].(string)
		if !ok || role != "admin" {
			// Warn может засорять систему логирования
			s.logger.Info("🚫 Access Restricted: Invalid Role",
				zap.Any("uid", claims["sub"]),
				zap.String("role_found", role),
			)
			s.respondError(w, http.StatusForbidden, "Доступ запрещен: требуются права администратора")
			return
		}

		// 5. Работа с UserID (поле "sub")
		sub, ok := claims["sub"].(string)
		if !ok {
			s.respondError(w, http.StatusUnauthorized, "ID пользователя не найден в токене")
			return
		}

		userID, err := uuid.Parse(sub)
		if err != nil {
			s.logger.Error("❌ UUID Parse Error from Token", zap.String("sub", sub), zap.Error(err))
			s.respondError(w, http.StatusUnauthorized, "Некорректный формат ID в системе")
			return
		}

		// 6. Передаем ID через типизированный контекст
		ctx := context.WithValue(r.Context(), types.UserIDKey, userID)
		ctx = context.WithValue(ctx, types.UserRoleKey, role)

		// Лог успешного входа (опционально для дебага)
		s.logger.Debug("👤 Authenticated", zap.String("uid", userID.String()))

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) RoleMiddleware(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Достаем роль, которую положил AuthMiddleware
			userRole, _ := r.Context().Value(types.UserRoleKey).(string)

			for _, role := range allowedRoles {
				if userRole == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			s.logger.Warn("🚫 Role access denied", zap.String("role", userRole))
			s.respondError(w, http.StatusForbidden, "У вас недостаточно прав")
		})
	}
}

// ZapLogger внедряет Uber Zap в цепочку chi.
func ZapLogger(log *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if viper.GetBool("server.debug") {
				log.Debug("incoming request",
					zap.String("method", r.Method),
					zap.Any("headers", r.Header))
			}
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			t1 := time.Now()

			defer func() {
				log.Info("request completed",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.Int("status", ww.Status()),
					zap.Duration("lat", time.Since(t1)),
					zap.String("req_id", middleware.GetReqID(r.Context())),
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
