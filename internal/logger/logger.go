package logger

import (
	"log"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Get настраивает и возвращает инстанс Zap логгера.
func Get() *zap.Logger {
	var config zap.Config
	if viper.GetString("env") == "development" {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}

	// Отключаем стек-трейс для Warn
	config.DisableStacktrace = false // оставляем включенным в целом

	logger, err := config.Build(zap.AddStacktrace(zap.ErrorLevel)) // оставляем только для Error и выше
	if err != nil {
		log.Fatalf("❌ FATAL: Failed to build logger: %v", err)
	}

	return logger
}
