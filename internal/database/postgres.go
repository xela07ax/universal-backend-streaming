package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"
	"github.com/xela07ax/universal-backend-streaming/internal/discovery"
	"go.uber.org/zap"
)

// dbTraceLogger — структура для реализации интерфейса pgx.QueryTracer
type dbTraceLogger struct {
	logger *zap.Logger
}

func (d *dbTraceLogger) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	d.logger.Info("[SQL EXEC]",
		zap.String("sql", data.SQL),
		zap.Any("args", data.Args))
	return ctx
}

func (d *dbTraceLogger) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	if data.Err != nil {
		d.logger.Error("[SQL ERROR]", zap.Error(data.Err))
	}
}

// NewPostgresConn создает пул соединений с PostgreSQL
func NewPostgresConn(sd discovery.ServiceDiscovery, logger *zap.Logger) (*pgxpool.Pool, error) {
	// 1. Строим DSN (внутри живет логика переключения Local/Discovery)
	dsn, err := BuildDSN(sd, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to build dsn: %w", err)
	}

	// 3. Настраиваем конфигурацию пула pgx
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse dsn: %w", err)
	}

	// РЕАЛИЗАЦИЯ DB_DEBUG Внедряем Zap в колбэк после подключения
	// Если флаг включен, логируем каждое новое соединение и настраиваем трейсинг
	isDebug := viper.GetBool("server.debug")

	if isDebug {
		// Включаем трейсинг самих SQL запросов
		config.ConnConfig.Tracer = &dbTraceLogger{logger: logger}
	}

	// Извлекаем хост и сервис для красивого логирования
	// pgxpool.Config позволяет легко достать эти данные из распарсенного DSN
	//connHost := config.ConnConfig.Host
	//dbService := viper.GetString("database.service_name")

	// Настраиваем AfterConnect с использованием внешних переменных
	dbServiceName := viper.GetString("database.service_name")
	targetHost := config.ConnConfig.Host
	// DEBUG логировать хост при каждом новом коннекте
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		if isDebug {
			logger.Info("[SQL DEBUG] Connection established",
				zap.String("service", dbServiceName),
				zap.String("host", targetHost))
		}
		return nil
	}

	// Настройки High-Load из Viper
	config.MaxConns = int32(viper.GetInt("database.max_conns"))
	config.MinConns = int32(viper.GetInt("database.min_conns"))
	config.MaxConnLifetime = viper.GetDuration("database.max_conn_lifetime")
	config.MaxConnIdleTime = 5 * time.Minute

	// 4. Создаем пул с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	// 5. Проверка физического соединения (Ping)
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("database ping failed on %s:%d: %w", targetHost, config.ConnConfig.Port, err)
	}

	logger.Info("Successfully connected to PostgreSQL",
		zap.Int32("max_conns", config.MaxConns),
	)

	return pool, nil
}

// BuildDSN «умный» строитель с поддержкой Test/Local и Service Discovery
func BuildDSN(sd discovery.ServiceDiscovery, logger *zap.Logger) (string, error) {
	var host string
	var port int
	dbServiceName := viper.GetString("database.service_name")

	// Пытаемся получить сервис через Discovery
	conf, err := sd.GetService(dbServiceName)

	if err == nil && conf.Host != "" {
		// КЕЙС 1: Нашли в Consul
		host = conf.Host
		port = conf.Port
		logger.Info("📡 Discovery SUCCESS",
			zap.String("service", dbServiceName),
			zap.String("actual_ip", host))
	} else if errors.Is(err, discovery.ErrDiscoveryDisabled) {
		// КЕЙС 2: Discovery выключен (Нормальный Static Mode)
		host = viper.GetString("database.host")
		port = viper.GetInt("database.port")
		logger.Info("🏠 Infrastructure: Using STATIC config", zap.String("host", host))
	} else {
		// КЕЙС 3: Discovery включен, но в Consul ПУСТО или ОШИБКА
		// ПЕРЕВОДИМ ИЗ ERROR В WARN + FALLBACK
		host = viper.GetString("database.host")
		port = viper.GetInt("database.port")

		logger.Warn("⚠️ Service Discovery: service not found, using fallback static host",
			zap.String("service", dbServiceName),
			zap.Error(err),
			zap.String("fallback_host", host))
	}
	// Финальная проверка на валидность хоста (защита от "lookup db-service", чтобы не было пустых строк)
	if host == "" {
		return "", fmt.Errorf("❌ FATAL: database host is empty (check your config or discovery)")
	}
	if port == 0 {
		port = 5432
	}

	// Сборка DSN
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		viper.GetString("database.user"),
		viper.GetString("database.password"),
		host, port,
		viper.GetString("database.name"),
		viper.GetString("database.sslmode"),
	), nil
}
