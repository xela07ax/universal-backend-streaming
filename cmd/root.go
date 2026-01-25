package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// RootCmd — это базовая команда Hydro Engine.
var RootCmd = &cobra.Command{
	Use:   "hydro",
	Short: "Hydro: Universal High-Load Streaming Framework",
	Long: `Hydro Engine — это универсальный бэкенд-каркас для стриминга медиа, 
построенный на принципах 12-Factor App и высокопроизводительной архитектуре Go.`,
}

// Execute Добавляет все дочерние команды к команде root и устанавливает соответствующие флаги.
// Эта функция вызывается из main.main(). Для команды rootCmd это должно произойти только один раз.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// 1. Глобальный флаг пути к конфигу
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is configs/hydro.yaml)")

	// 2. Настройка флага дебага БД
	// Имя флага в CLI: --database.debug (соответствует иерархии конфига)
	RootCmd.PersistentFlags().Bool("database.debug", false, "log sql queries to console")

	// Привязываем флаг к Viper. Если флаг передан, он перезапишет значение из YAML.
	if err := viper.BindPFlag("database.debug", RootCmd.PersistentFlags().Lookup("database.debug")); err != nil {
		// Используем Fatal, так как неверная инициализация флагов — это баг разработки
		log.Fatalf("❌ FATAL: database.debug flag binding failed: %v", err)
	}

	// 3. Настройка флага дебага API (аналогично)
	RootCmd.PersistentFlags().Bool("server.debug", false, "log detailed http requests")
	if err := viper.BindPFlag("server.debug", RootCmd.PersistentFlags().Lookup("server.debug")); err != nil {
		log.Fatalf("❌ FATAL: server.debug flag binding failed: %v", err)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, _ := os.UserHomeDir()
		// 1. Ищем в домашней папке (~/.hydro.yaml)
		viper.AddConfigPath(home)
		// 2. Ищем в папке с конфигами проекта (./configs/config.yaml)
		viper.AddConfigPath("configs")

		// Обновленное имя конфига — hydro
		viper.SetConfigName("hydro") // Важно! Имя должно совпадать с файлом
		viper.SetConfigType("yaml")
	}

	viper.SetEnvPrefix("HYDRO")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		if viper.GetBool("server.debug") {
			fmt.Printf("Hydro Engine: Using config [%s]\n", viper.ConfigFileUsed())
		}
	}
}
