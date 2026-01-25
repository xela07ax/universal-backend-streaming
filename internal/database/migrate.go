package database

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // Драйвер для БД
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/xela07ax/universal-backend-streaming/internal/discovery"
	"go.uber.org/zap"
)

// ApplyMigrations управляет схемой БД в зависимости от переданного действия (up, down, status)
func ApplyMigrations(sd discovery.ServiceDiscovery, logger *zap.Logger, action string) error {
	dsn, err := BuildDSN(sd, logger)
	if err != nil {
		return fmt.Errorf("migration dsn build failed: %w", err)
	}

	// Создаем драйвер из встроенных файлов (Embed FS)
	d, err := iofs.New(MigrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create iofs driver: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, dsn)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	// Гарантируем закрытие соединений мигратора
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil || dbErr != nil {
			logger.Warn("Migration: error closing connections", zap.Error(srcErr), zap.Error(dbErr))
		}
	}()

	logger.Info("Database Migration", zap.String("action", action), zap.String("source", "EmbedFS"))

	switch action {
	case "up":
		if err := m.Up(); err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				logger.Info("✅ Database is up to date (no changes)")
				return nil
			}
			return fmt.Errorf("up migration failed: %w", err)
		}
	case "down":
		// Откатываем на 1 шаг назад
		if err := m.Steps(-1); err != nil {
			return fmt.Errorf("down migration failed: %w", err)
		}
	case "status":
		version, dirty, err := m.Version()
		if err != nil {
			if errors.Is(err, migrate.ErrNilVersion) {
				logger.Info("🔍 Status: Database is empty (version 0)")
				return nil
			}
			return fmt.Errorf("failed to get migration version: %w", err)
		}
		logger.Info("🔍 Database Status",
			zap.Uint("version", version),
			zap.Bool("dirty", dirty))
	default:
		return fmt.Errorf("unsupported migration action: %s", action)
	}

	logger.Info("✅ Migration task finished", zap.String("action", action))
	return nil
}
