package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"github.com/xela07ax/universal-backend-streaming/internal/discovery"
	"go.uber.org/zap"
)

func NewRedisClient(sd discovery.ServiceDiscovery, logger *zap.Logger) (*redis.Client, error) {
	var host string
	var port int
	redisServiceName := viper.GetString("redis.service_name")

	// Пытаемся получить адрес через Discovery
	conf, err := sd.GetService(redisServiceName)

	if err == nil && conf.Host != "" {
		// КЕЙС 1: Discovery включен и успешно нашел Redis
		host = conf.Host
		port = conf.Port
		logger.Info("📡 Redis Discovery: SUCCESS", zap.String("host", host))
	} else if errors.Is(err, discovery.ErrDiscoveryDisabled) {
		// КЕЙС 2: Discovery выключен (Static Mode)
		host = viper.GetString("redis.host")
		port = viper.GetInt("redis.port")
		logger.Info("🏠 Redis: using STATIC config", zap.String("host", host))
	} else {
		// КЕЙС 3: Discovery включен, но произошла ошибка (Consul упал или сервис не найден)
		// Для Redis (как и для видео) мы используем мягкий Fallback на статику
		host = viper.GetString("redis.host")
		port = viper.GetInt("redis.port")
		logger.Warn("⚠️ Redis Discovery: error, using fallback config",
			zap.String("service", redisServiceName),
			zap.Error(err),
			zap.String("fallback_host", host))
	}

	// Дефолты 2026, если данных нет ни в Discovery, ни в конфиге
	if host == "" {
		host = "localhost"
	}
	if port == 0 {
		port = 6379
	}

	addr := fmt.Sprintf("%s:%d", host, port)

	// Инициализация клиента go-redis (v9)
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: viper.GetString("redis.password"),
		DB:       viper.GetInt("redis.db"),
	})

	// Проверка связи (Ping)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed at %s: %w", addr, err)
	}

	return rdb, nil
}
