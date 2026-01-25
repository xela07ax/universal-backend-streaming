package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// MediaAsset представляет структуру записи в таблице media_assets
type MediaAsset struct {
	ID          uuid.UUID              `json:"id"`
	OwnerID     uuid.UUID              `json:"owner_id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Status      string                 `json:"status"`
	StoragePath string                 `json:"storage_path"`
	Metadata    map[string]interface{} `json:"metadata"` // JSONB передается как map, и pgx сам конвертирует его в JSON для Postgres.
}

// MediaRepository предоставляет методы для работы с БД
// Она инкапсулирует все SQL-запросы к таблице media_assets и управляет
// жизненным циклом транзакций через переданный контекст. Предоставляет методы для работы с БД

// DBTX — общий интерфейс для пула, транзакции или мока, описывает минимальный набор методов для работы с Postgres, используем стандарт методов, который понимает pgxmock v3
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row
}

type MediaRepository struct {
	db DBTX // Используем интерфейс вместо *pgxpool.Pool
}

// NewMediaRepository создает новый экземпляр репозитория
func NewMediaRepository(db DBTX) *MediaRepository {
	return &MediaRepository{db: db}
}

// SaveAsset сохраняет метаданные видео в базу данных
func (r *MediaRepository) SaveAsset(ctx context.Context, asset *MediaAsset) error {
	query := `
		INSERT INTO media_assets (owner_id, title, description, storage_path, status, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	// Выполняем запрос с использованием пула соединений
	err := r.db.QueryRow(ctx, query,
		asset.OwnerID,
		asset.Title,
		asset.Description,
		asset.StoragePath,
		asset.Status,
		asset.Metadata,
	).Scan(&asset.ID, nil) // Получаем сгенерированный базой UUID обратно

	if err != nil {
		return fmt.Errorf("repository: failed to save asset: %w", err)
	}

	return nil
}

// GetAllAssets возвращает список всех медиа-файлов из базы данных.
func (r *MediaRepository) GetAllAssets(ctx context.Context) ([]MediaAsset, error) {
	query := `SELECT id, owner_id, title, description, status, storage_path, metadata FROM media_assets ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("repository: failed to fetch assets: %w", err)
	}
	defer rows.Close()

	var assets []MediaAsset
	for rows.Next() {
		var a MediaAsset
		if err := rows.Scan(&a.ID, &a.OwnerID, &a.Title, &a.Description, &a.Status, &a.StoragePath, &a.Metadata); err != nil {
			return nil, err
		}
		assets = append(assets, a)
	}
	return assets, nil
}

// GetAssetByID находит запись о медиа-активе по его UUID.
// Используется для получения путей к файлам перед генерацией ссылки в VideoProvider.
func (r *MediaRepository) GetAssetByID(ctx context.Context, id uuid.UUID) (*MediaAsset, error) {
	query := `
		SELECT id, owner_id, title, description, status, storage_path, metadata 
		FROM media_assets 
		WHERE id = $1 
		LIMIT 1
	`

	var asset MediaAsset
	// Выполняем запрос через пул соединений pgx
	err := r.db.QueryRow(ctx, query, id).Scan(
		&asset.ID,
		&asset.OwnerID,
		&asset.Title,
		&asset.Description,
		&asset.Status,
		&asset.StoragePath,
		&asset.Metadata,
	)

	if err != nil {
		return nil, fmt.Errorf("repository: asset not found: %w", err)
	}

	return &asset, nil
}
