ARG GO_VERSION=1.25.5

# --- ЭТАП 1: Сборка Фронтенда (Vue 3) ---
FROM node:20-alpine AS ui-builder
WORKDIR /app/web
# Копируем только файлы зависимостей для кэширования
COPY web/package*.json ./
RUN npm install
# Копируем исходники и собираем
COPY web/ .
RUN npm run build

# --- ЭТАП 2: Сборка Бэкенда (Go 1.25.5) ---
FROM golang:${GO_VERSION}-alpine AS backend-builder
WORKDIR /app
# Устанавливаем зависимости для сборки (gcc нужен, если используется CGO,
# но мы собираем статический бинарник)
RUN apk add --no-cache git ca-certificates
# Копируем файлы зависимостей
COPY go.mod go.sum ./
RUN go mod download
# Копируем исходный код
COPY . .
# Сборка оптимизированного бинарника для Linux
# -ldflags="-w -s" убирает отладочную информацию, уменьшая размер файла на 25%
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o hydro main.go

# --- ЭТАП 3: Финальный минималистичный образ ---
FROM alpine:3.19
WORKDIR /app

# Устанавливаем временную зону и сертификаты для сетевых запросов (Postgres/Consul)
RUN apk add --no-cache ca-certificates tzdata

# Копируем бинарник из этапа 2
COPY --from=backend-builder /app/hydro .
# Копируем собранный фронтенд из этапа 1 в папку, которую ожидает Go
COPY --from=ui-builder /app/web/dist ./web/dist

# Создаем папку для загрузки видео
RUN mkdir -p web/dist/uploads

# Открываем порт (совпадает с нашим config.yaml)
EXPOSE 8080

# Запуск Hydro Engine с указанием конфига через переменную окружения
# Переменную можно переопределить в docker-compose
ENV HYDRO_CONFIG=configs/hydro.yaml
CMD ["./hydro", "serve", "--config", "configs/hydro.yaml"]
