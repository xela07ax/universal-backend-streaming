# --- Параметры Hydro Engine 2026 ---
APP_NAME := hydro
BIN_DIR  := bin
TARGET   := $(BIN_DIR)/$(APP_NAME)
CONF_DEV := configs/hydro.yaml
CONF_PROD:= configs/production.yaml

# Минимально необходимая версия Go (1.25.5+)
GO_MIN_VERSION := 1.25
GO_VERSION_CHECK := $(shell go version | sed -re 's/.*go([0-9]+)\.([0-9]+)\.?([0-9]*).*/\1\2\3/' | cut -c1-3)
GO_MIN_INT := 125

# --- Основные команды ---

.PHONY: all build clean dev migrate-up check-env check-go help lint test test-coverage docs-view test test-race cover

all: build

check-go: ## Проверить версию Go (требуется >= 1.25.5)
	@if [ $(GO_VERSION_CHECK) -lt $(GO_MIN_INT) ]; then \
		echo "❌ ОШИБКА: Требуется Go >= $(GO_MIN_VERSION). У вас: $$(go version)"; exit 1; \
	else \
		echo "✅ Go version check: OK"; \
	fi

build: check-go ## Собрать бинарник в папку bin
	@mkdir -p $(BIN_DIR)
	@echo "🏗️  Сборка $(APP_NAME) в $(BIN_DIR)..."
	go build -ldflags="-w -s" -o $(TARGET) main.go
	@echo "✨ Сборка завершена: $(TARGET)"

dev: build ## Запуск
	@echo "🧹 Очистка порта 8080..."
	@taskkill /F /IM $(APP_NAME).exe /T 2>nul || true
	@echo "🚀 Запуск Hydro в режиме DEVELOPMENT..."
	./$(TARGET) serve --config $(CONF_DEV) --server.debug --database.debug

check-env: build ## Проверить текущий резолвинг (Consul vs Static)
	@echo "🔍 Проверка конфигурации системы..."
	./$(TARGET) check-env --config $(CONF_DEV)

# --- Работа с миграциями ---
migrate-up: build ## Применить все миграции на базу (192.168.72.37)
	@echo "🚀 Запуск миграций..."
	./$(TARGET) migrate --config $(CONF_DEV) --action up

migrate-down: build ## Откатить последнюю миграцию
	@echo "⚠️  Откат миграции..."
	./$(TARGET) migrate --config $(CONF_DEV) --action down

migrate-status: build ## Проверить текущую версию схемы БД
	./$(TARGET) migrate --config $(CONF_DEV) --action status

# --- Работа с инфраструктурой ---

consul-reg: ## Регистрация Postgres (192.168.72.37) в удаленном Consul
	@curl --header "X-Consul-Token: hydro-admin-token-2026" \
		--request PUT \
		--data '{"ID": "db-1", "Name": "db-service", "Address": "192.168.72.37", "Port": 5432}' \
		http://localhost:8500/v1/agent/service/register
	@echo "✅ База данных зарегистрирована в Service Discovery"

redis-cli: ## Зайти в консоль Redis для проверки сессий
	docker exec -it hydro-redis redis-cli -a "hydro-pass-2026"

# --- Очистка ---

clean: ## Удалить бинарники и временные файлы сборки
	@rm -rf $(BIN_DIR)
	@rm -rf web/dist
	@go clean -cache
	@echo "🧹 Папка bin/ и кэш очищены"

help: ## Показать список всех команд
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# --- Контроль качества и Тестирование ---

lint: ## Запустить статический анализ кода
	@echo "🔍 Running golangci-lint..."
	@golangci-lint run ./...

test-coverage: ## Проверить покрытие кода тестами
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Отчет о покрытии создан: coverage.html"

docs-view: ## Показать интерактивную карту API и параметры запросов
	@echo "--- 📚 Hydro Engine API Documentation (2026) ---"
	@if command -v jq >/dev/null; then \
		curl -s http://localhost:8080/api/v1/docs | jq -r '.data.routes[] | \
		"\033[1;32m[\(.method)]\033[0m \033[1;34m\(.path)\033[0m\n" + \
		"  📝 Описание:  \(.description)\n" + \
		"  🔒 Защищен:   \(if .protected then "✅ Да" else "❌ Нет" end)\n" + \
		"  📥 Параметры: \(if .params then "\n    " + (.params | to_entries | map("\(.key): \(.value)") | join("\n    ")) else "-" end)\n" + \
		"  📦 Body:      \(if .body then "\n    " + (.body | to_entries | map("\(.key): \(.value)") | join("\n    ")) else "-" end)\n"'; \
	else \
		echo "⚠️  Подсказка: Установите 'jq' для цветного вывода. Сейчас выводится RAW JSON:"; \
		curl -s http://localhost:8080/api/v1/docs; \
	fi
	@echo "\n------------------------------------------------"

test: build ## Запустить быстрые unit-тесты
	@echo "🧪 Running unit tests..."
	go test -v ./internal/...

test-race: ## Проверить состояние гонки (Race Condition) - важно для стриминга
	@echo "🏎️  Checking for race conditions..."
	go test -race -v ./internal/...

cover: ## Создать визуальный отчет о покрытии кода тестами
	@go test -coverprofile=coverage.out ./internal/...
	@go tool cover -html=coverage.out -o bin/coverage.html
	@echo "📊 Report saved to bin/coverage.html"
