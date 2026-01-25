package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/xela07ax/universal-backend-streaming/internal/api"
	"github.com/xela07ax/universal-backend-streaming/internal/database"
	"github.com/xela07ax/universal-backend-streaming/internal/discovery"
	"github.com/xela07ax/universal-backend-streaming/internal/logger"
	"github.com/xela07ax/universal-backend-streaming/internal/streaming"
	"go.uber.org/zap"

	"log"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Запуск API сервера Hydro Engine",
	// Вместо анонимной функции используем ссылку на именованную
	Run: runServe,
}

// runServe вынесена отдельно, чтобы избежать конфликтов области видимости
func runServe(cmd *cobra.Command, args []string) {
	// Вывод приветственного сообщения и метаданных конфига
	fmt.Println(`
    __  統領 Hydro Engine
   / / / /_  __/ __ \____ 
  / /_/ / / / / /_/ / __ \
 / __  / /_/ / _, _/ /_/ /
/_/ /_/\__, /_/ |_|\____/ 
      /____/`)
	// 1. Инициализируем логгер ОДИН раз (используем нашу новую функцию)
	l := logger.Get()
	defer func() { _ = l.Sync() }()
	l.Info("🚀 Hydro Engine Starting...")

	l.Info("Starting Hydro Server",
		zap.String("version", "2026.1"),
		zap.String("config_source", viper.ConfigFileUsed()), // Показывает путь к файлу
		zap.String("env", viper.GetString("env")),
		zap.String("addr", ":"+viper.GetString("server.port")),
	)

	// 2. Инициализируем ConfigResolver для сетевой гибкости (Local/Docker)
	registry := viper.GetStringMapString("discovery.services")
	resolver := discovery.NewConfigResolver()
	// Выведем куда мы на самом деле стучимся по сети
	l.Info("Service Discovery initialized",
		zap.Any("services", registry),
	)
	// 3. Подключение к базе данных PostgreSQL (через pgxpool)
	// Передаем резолвер, чтобы БД знала, куда подключаться
	db, err := database.NewPostgresConn(resolver, l)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}

	// 4. Инициализация VideoProvider для генерации URL
	videoProvider, err := streaming.NewVideoProvider(resolver, l)
	if err != nil {
		l.Fatal("video provider init failed", zap.Error(err))
	}

	// 1. Подключаем Redis
	rdb, err := database.NewRedisClient(resolver, l)
	if err != nil {
		l.Fatal("Failed to initialize Redis", zap.Error(err))
	}
	l.Info("Connected to Redis", zap.String("addr", rdb.Options().Addr))

	// 2. Создание и запуск API сервера
	server, _ := api.NewServer(db, rdb, videoProvider, l, viper.GetString("auth.jwt_secret"))

	if err != nil {
		l.Fatal("api server init failed", zap.Error(err))
	}
	addr := ":" + viper.GetString("server.port")

	// Graceful Shutdown
	// 1. Создаем канал для перехвата сигналов ОС (Ctrl+C, kill)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// 2. Запускаем сервер в отдельной горутине
	go func() {
		l.Info("Hydro Server start", zap.String("addr", addr))
		if err := server.Start(addr); err != nil && err != http.ErrServerClosed {
			l.Fatal("Hydro Server failed to start", zap.Error(err))
		}
	}()
	// 3. Блокируем выполнение, пока не придет сигнал (Ctrl+C)
	sig := <-stop
	l.Info("Shutdown signal received", zap.String("signal", sig.String()))

	// 4. Настраиваем дедлайн для завершения (15 секунд в 2026 году — золотой стандарт)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// ПЕРВЫМ делом: Останавливаем HTTP сервер (он перестает принимать новые коннекты)
	if err := server.Shutdown(shutdownCtx); err != nil {
		l.Error("HTTP shutdown error", zap.Error(err))
	}

	// ВТОРЫМ делом: Закрываем базу данных
	// Это гарантирует, что активные транзакции из п.1 успели дойти до БД
	server.Close()

	l.Info("Hydro Engine stopped gracefully")
}

func init() {
	RootCmd.AddCommand(serveCmd)

	// Дефолтные настройки сервера
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("env", "production")
	// --- Настройки безопасности ---
	viper.SetDefault("auth_login_token_length", 8)
	viper.SetDefault("auth_login_token_expiry", "11m")
	viper.SetDefault("auth_jwt_secret", "random_secure_string_2026")

	// Настройки пула соединений
	viper.SetDefault("database.max_conns", 25)
	viper.SetDefault("database.min_conns", 5)
	viper.SetDefault("database.max_conn_lifetime", "30m")

	// --- Системные настройки (ДЛЯ СТРИМИНГА) ---
	viper.SetDefault("video.service_name", "video-storage")
	viper.SetDefault("video.port", 8080)

	// Мапинг для резолвера (пустой по умолчанию для Docker DNS)
	viper.SetDefault("discovery.services", map[string]string{})
}
