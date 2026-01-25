/*
Package api реализует транспортный слой Hydro Engine.

Использование go-chi в 2026 году — это золотая середина для стриминг-бойлерплейта.
В отличие от тяжелых фреймворков (типа Gin или Fiber), chi полностью совместим
со стандартной библиотекой net/http. Для Hydro это критически важно по трем причинам:

1. Производительность стриминга: chi не создает лишних аллокаций на куче при парсинге роутов,
что крайне важно для передачи тяжелого видео-трафика и минимизации задержек.

2. Контексты (Context-friendly): Он идеально работает с context.Context, который уже внедрен
в наши репозитории и провайдеры для управления жизненным циклом запроса.

3. Middleware: У chi лучший механизм цепочек middleware (логирование, авторизация, RealIP),
который не ломает стандартные хендлеры Go и позволяет легко масштабировать функционал.
*/
package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"github.com/xela07ax/universal-backend-streaming/internal/repository"
	"github.com/xela07ax/universal-backend-streaming/internal/streaming"
	"go.uber.org/zap"
)

// Server — основной узел Hydro Engine.
// Он объединяет транспорт (HTTP), хранилище (DB) и бизнес-логику.
type Server struct {
	router     *chi.Mux
	httpServer *http.Server
	logger     *zap.Logger
	db         *pgxpool.Pool
	rdb        *redis.Client
	media      *repository.MediaRepository
	users      *repository.UserRepository
	video      *streaming.VideoProvider
	// ... ваши репозитории (media и т.д.)
	// Секрет для JWT берем из конфига через Viper
	jwtSecret string
}

// NewServer собирает сервер и настраивает все зависимости.
func NewServer(db *pgxpool.Pool, rdb *redis.Client, vp *streaming.VideoProvider, log *zap.Logger, secret string) (*Server, error) {
	s := &Server{
		router:    chi.NewRouter(),
		logger:    log,
		db:        db,
		rdb:       rdb,
		video:     vp,
		jwtSecret: secret,
		users:     repository.NewUserRepository(db),
		media:     repository.NewMediaRepository(db),
	}

	s.setupRoutes()
	return s, nil
}

// Start запускает HTTP сервер.
func (s *Server) Start(addr string) error {
	// Создаем папку для хранения, если её нет
	if err := os.MkdirAll(s.video.GetBasePath(), 0755); err != nil {
		s.logger.Error("Failed to create storage directory", zap.Error(err))
	}

	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.router,
		// Важно ставить таймауты, чтобы сокеты не висели вечно (зомби-процесс)
		ReadTimeout:  30 * time.Minute, // Даем время на загрузку тяжелых видео
		WriteTimeout: 30 * time.Minute,
		IdleTimeout:  120 * time.Second,
	}

	// Выводим отчет перед запуском
	s.printStartupReport(addr)

	return s.httpServer.ListenAndServe()
}

// Shutdown позволяет изящно остановить сервер, не обрывая активные соединения.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down Hydro HTTP server...")
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) setupRoutes() {
	// 1. Глобальные Middleware (Observed & Safe)
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(ZapLogger(s.logger))
	s.router.Use(middleware.Recoverer)
	s.router.Use(s.setupCORS().Handler)

	// --- 2. API РОУТЫ ---
	s.router.Route("/api/v1", func(r chi.Router) {
		// --- Публичная зона (просмотр видео) ---
		r.Group(func(r chi.Router) {
			r.Get("/docs", s.handleGetDocs) // Публичная документация
			r.Get("/video/{id}", s.handleGetVideoURL)
			r.Get("/health", s.handleHealth)
			r.Post("/login", s.handleLogin)
			r.Post("/refresh", s.handleRefresh)

		})

		// --- Приватная зона (Админ панель) ---
		r.Group(func(r chi.Router) {
			r.Use(s.AuthMiddleware) // Защищаем всю группу
			r.Post("/admin/upload", s.handleAdminUploadAsset)
			r.Get("/admin/assets", s.handleAdminListAssets)
			r.Post("/admin/assets", s.handleAdminCreateAsset)
			r.Post("/logout", s.handleLogout)
		})
	})

	// 3. Раздача статических файлов фронтенда из папки web/dist
	staticPath := "./web/dist"

	s.router.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
		// 1. Формируем ПОЛНЫЙ путь к файлу на диске
		// filepath.FromSlash гарантирует правильные слеши на Windows (\) и Linux (/)
		path := filepath.Join(staticPath, filepath.Clean(r.URL.Path))

		// 2. Проверяем, существует ли такой файл физически
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			http.ServeFile(w, r, path)
			return
		}

		// 3. Если это не файл (а роут Vue, например /admin/dashboard),
		// отдаем index.html для работы SPA History Mode
		http.ServeFile(w, r, filepath.Join(staticPath, "index.html"))
	})
}

func (s *Server) setupCORS() *cors.Cors {
	// Считываем список из конфига (вернет пустой слайс, если ключа нет)
	allowedOrigins := viper.GetStringSlice("server.cors.allowed_origins")

	// Если включен флаг allow_local — добавляем локальные адреса динамически
	if viper.GetBool("server.cors.allow_local") {
		port := viper.GetString("server.port")

		// Формируем суффикс порта: если 80, 443 или пусто — суффикс пустой
		portSuffix := ""
		if port != "" && port != "80" && port != "443" {
			portSuffix = ":" + port
		}

		// Формируем список локальных адресов
		localOrigins := []string{
			"http://localhost" + portSuffix,
			"http://127.0.0.1" + portSuffix,
		}
		allowedOrigins = append(allowedOrigins, localOrigins...)
	}

	// Удаляем дубликаты для чистоты отчета
	allowedOrigins = uniqueStrings(allowedOrigins)

	return cors.New(cors.Options{
		AllowedOrigins: allowedOrigins, // Используем динамический список из YAML
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		// Для видео-стриминга обязательно прокидываем заголовки диапазонов
		ExposedHeaders:   []string{"Link", "Content-Range", "Accept-Ranges", "Content-Length"},
		AllowCredentials: true,
		MaxAge:           300,
		Debug:            viper.GetBool("server.debug"),
	})
}

// GetRouter возвращает экземпляр chi роутера для генерации документации.
func (s *Server) GetRouter() *chi.Mux {
	return s.router
}

// Close завершает работу со всеми внешними ресурсами (Postgres, Redis).
func (s *Server) Close() {
	s.logger.Info("Starting graceful shutdown of data sources...")

	// 1. Закрываем Postgres
	if s.db != nil {
		s.db.Close()
		s.logger.Info("PostgreSQL connection pool closed")
	}

	// 2. Закрываем Redis
	if s.rdb != nil {
		if err := s.rdb.Close(); err != nil {
			s.logger.Error("Failed to close Redis connection", zap.Error(err))
		} else {
			s.logger.Info("Redis connection closed successfully")
		}
	}
}

func (s *Server) printStartupReport(addr string) {
	// 1. Извлекаем реальный хост БД из пула pgxpool
	realDBHost := "disconnected"
	realDBName := "unknown"
	if s.db != nil {
		// В pgx v5 и выше конфиг содержит все итоговые параметры
		dbCfg := s.db.Config().ConnConfig
		realDBHost = fmt.Sprintf("%s:%d", dbCfg.Host, dbCfg.Port)
		realDBName = dbCfg.Database
	}

	// 2. Извлекаем реальный хост Redis из клиента go-redis
	realRedisHost := "disconnected"
	if s.rdb != nil {
		realRedisHost = s.rdb.Options().Addr
	}

	// 3. Определяем режим Discovery
	discoveryStatus := "OFF (Static)"
	if viper.GetBool("discovery.enabled") {
		discoveryStatus = fmt.Sprintf("ON (Consul: %s)", viper.GetString("discovery.consul_addr"))
	}

	// 4. Печатаем правдивый отчет
	s.logger.Info("🚀 HYDRO ENGINE STARTUP REPORT",
		zap.String("version", "2026.1.0"),
		zap.String("api_addr", addr),
		zap.Strings("cors_allowed", viper.GetStringSlice("server.cors.allowed_origins")),
		zap.String("mode", viper.GetString("env")),
		zap.String("discovery", discoveryStatus),
		zap.String("actual_db", realDBHost),          // Реально зарезолвленный хост
		zap.String("db_name", realDBName),            // Реальное имя базы
		zap.String("actual_redis", realRedisHost),    // Реальный адрес Redis
		zap.String("storage", s.video.GetBasePath()), // Реальный адрес Redis
		zap.String("video_host", s.video.GetHost()),
		zap.Int("video_port", s.video.GetPort()),
		zap.String("storage", s.video.GetBasePath()),
	)

	if viper.GetBool("database.debug") {
		s.logger.Warn("⚠️  SQL TRACE ACTIVE: Performance may be affected")
	}
}

// uniqueStrings удаляет дубликаты из среза строк, сохраняя порядок первого появления.
func uniqueStrings(input []string) []string {
	if len(input) == 0 {
		return input
	}

	// Используем пустую структуру struct{}, так как она не занимает памяти в map
	keys := make(map[string]struct{})
	result := make([]string, 0, len(input))

	for _, entry := range input {
		// Если строки еще нет в карте — добавляем в результат
		if _, exists := keys[entry]; !exists {
			keys[entry] = struct{}{}
			result = append(result, entry)
		}
	}

	return result
}
