package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xela07ax/universal-backend-streaming/internal/database"
	"github.com/xela07ax/universal-backend-streaming/internal/discovery"
	"github.com/xela07ax/universal-backend-streaming/internal/logger"
	"go.uber.org/zap"
)

var reset bool

// Переменная для хранения действия (up/down/status)
var migrationAction string

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Управление миграциями базы данных",
	Run: func(cmd *cobra.Command, args []string) {
		// 1. Инициализируем логгер через наш новый пакет internal/logger
		l := logger.Get()
		defer func() { _ = l.Sync() }()

		// 2. Инициализируем Stateless резолвер
		resolver := discovery.NewConfigResolver()

		l.Info("🚀 Hydro Migration Started",
			zap.String("action", migrationAction),
			zap.String("db_service", viper.GetString("database.service_name")),
		)

		// 3. Вызываем обновленный мигратор с передачей действия (action)
		if err := database.ApplyMigrations(resolver, l, migrationAction); err != nil {
			l.Fatal("❌ Migration failed", zap.Error(err))
		}

		l.Info("✅ Database migration completed successfully")
	},
}

func init() {
	// Добавляем команду в Root
	RootCmd.AddCommand(migrateCmd)

	// Регистрируем локальный флаг --action для команды migrate
	// По умолчанию ставим "up", как это принято в 2026 году
	migrateCmd.Flags().StringVar(&migrationAction, "action", "up", "Действие: up, down или status")
}

func init() {
	RootCmd.AddCommand(migrateCmd)
	migrateCmd.Flags().BoolVar(&reset, "reset", false, "Reset all data and run migrations from scratch")
}
