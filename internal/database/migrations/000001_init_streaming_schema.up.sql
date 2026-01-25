-- Включаем расширение для генерации UUID (в Postgres 13+ обычно активно по умолчанию)
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 1. Таблица пользователей
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(100) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
                             );

-- 2. Таблица медиа-контента (видео активы)
CREATE TABLE IF NOT EXISTS media_assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Внешний ключ: ссылаемся на id из таблицы users
    owner_id UUID NOT NULL,

    title VARCHAR(255) NOT NULL,
    description TEXT,

    -- Статус: 'processing', 'ready', 'failed'
    status VARCHAR(20) DEFAULT 'processing',

    -- Базовый путь в хранилище (используется нашим VideoProvider)
    storage_path TEXT NOT NULL,

    -- Длительность в секундах
    duration INT DEFAULT 0,

    -- Метаданные (кодеки, разрешение и т.д.)
    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Описание связи
    CONSTRAINT fk_media_owner
    FOREIGN KEY(owner_id)
    REFERENCES users(id)
     ON DELETE CASCADE
    );

-- 3. Таблица для стриминг-протоколов (HLS/DASH)
CREATE TABLE IF NOT EXISTS streaming_endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Ссылаемся на id из таблицы media_assets
    asset_id UUID NOT NULL,

    -- hls, dash
    protocol VARCHAR(10) NOT NULL,

    -- Путь к манифесту (напр. playlist.m3u8)
    manifest_path TEXT NOT NULL,

    -- 1080p, 720p и т.д.
    resolution VARCHAR(20),

    CONSTRAINT fk_endpoint_asset
    FOREIGN KEY(asset_id)
    REFERENCES media_assets(id)
    ON DELETE CASCADE
    );

-- Индексы для оптимизации поиска в 2026 году
CREATE INDEX IF NOT EXISTS idx_media_assets_owner ON media_assets(owner_id);
CREATE INDEX IF NOT EXISTS idx_media_assets_status ON media_assets(status);
CREATE INDEX IF NOT EXISTS idx_streaming_endpoints_asset ON streaming_endpoints(asset_id);

-- UUID: Поля id везде имеют тип UUID, что позволяет генерировать их на стороне Go (через google/uuid) и не ждать ответа базы.
-- JSONB: Используется для metadata, чтобы ты мог хранить там любую техническую информацию без изменения схемы.