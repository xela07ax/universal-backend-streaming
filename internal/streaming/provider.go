package streaming

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"github.com/xela07ax/universal-backend-streaming/internal/discovery"
	"go.uber.org/zap"
)

// VideoProvider отвечает за поиск видеофайлов в хранилище
// и подготовку их к стримингу.
type VideoProvider struct {
	// Базовый путь к папке с видео (например, "./web/dist/uploads")
	basePath string

	// Логгер для отслеживания ошибок чтения и доступа
	logger *zap.Logger

	// Параметры хоста (если видео раздается с другого узла)
	host string
	port int
}

// NewVideoProvider создает новый экземпляр VideoProvider.
// Он разрешает имя сервиса через переданный ServiceDiscovery и подготавливает финальный хост.
func NewVideoProvider(sd discovery.ServiceDiscovery, logger *zap.Logger) (*VideoProvider, error) {
	// 1. Сразу задаем дефолты из статического конфига
	host := viper.GetString("video.host")
	port := viper.GetInt("video.port")
	if host == "" {
		host = "localhost"
	}
	if port == 0 {
		port = 8080
	}

	serviceName := viper.GetString("video.service_name")

	// 2. Просто запрашиваем сервис. Резолвер сам проверит discovery.enabled.
	conf, err := sd.GetService(serviceName)
	if err == nil && conf.Host != "" {
		// Если Discovery включен и сервис найден
		host = conf.Host
		port = conf.Port
		logger.Info("📡 VideoProvider: resolved via Discovery", zap.String("host", host))
	} else if errors.Is(err, discovery.ErrDiscoveryDisabled) {
		// Если Discovery просто выключен — это нормальное поведение (Static Mode)
		logger.Info("🏠 VideoProvider: using STATIC config", zap.String("host", host))
	} else {
		// Если Discovery включен, но произошла реальная ошибка (Consul упал или сервис не найден)
		logger.Warn("⚠️ VideoProvider: discovery error, using fallback config",
			zap.String("service", serviceName),
			zap.Error(err),
			zap.String("fallback_host", host))
	}

	basePath := viper.GetString("video.storage_path")
	if basePath == "" {
		basePath = "./web/dist/uploads"
		logger.Warn("video.storage_path not set, using default", zap.String("path", basePath))
	}

	return &VideoProvider{
		basePath: basePath,
		host:     host,
		port:     port,
		logger:   logger,
	}, nil
}

// BuildURL генерирует полный URL для указанного роута, используя
// закешированное базовое состояние хоста.
func (p *VideoProvider) BuildURL(storagePath string) string {
	// Превращаем "uploads/my_video.mp4" -> "/api/v1/storage/my_video.mp4"
	// Мы убираем префикс папки "uploads/", так как роут в Chi уже указывает на неё
	cleanPath := strings.TrimPrefix(storagePath, "uploads/")
	return "/api/v1/storage/" + cleanPath
}

// GetBasePath возвращает текущий путь к хранилищу видео
func (p *VideoProvider) GetBasePath() string {
	if p == nil {
		return "not initialized"
	}
	return p.basePath
}

func (p *VideoProvider) GetHost() string {
	if p == nil {
		return "unknown"
	}
	return p.host
}

func (p *VideoProvider) GetPort() int {
	if p == nil {
		return 0
	}
	return p.port
}

// todo not use
func (p *VideoProvider) GetSafePath(fileName string) (string, error) {
	// 1. Очищаем путь от ".." и лишних слешей
	cleanPath := filepath.Clean(fileName)

	// 2. Формируем финальный путь
	finalPath := filepath.Join(p.basePath, cleanPath)

	// 3. Проверка: не "вылетел" ли путь за пределы basePath после Join?
	// В 2026 году это стандарт защиты от Path Traversal
	rel, err := filepath.Rel(p.basePath, finalPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("security alert: attempt to access outside directory: %s", fileName)
	}

	return finalPath, nil
}
