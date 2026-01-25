package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
)

func TestMediaRepository_SaveAsset(t *testing.T) {
	// 1. Создаем мок пула
	mock, err := pgxmock.NewPool()
	assert.NoError(t, err)
	defer mock.Close()

	// 2. Инициализируем репозиторий (теперь mock подходит под интерфейс DBTX)
	repo := NewMediaRepository(mock)

	// Создаем валидный UUID для теста
	testOwnerID := uuid.New()
	newAssetID := uuid.New()
	now := time.Now()

	asset := &MediaAsset{
		OwnerID:     testOwnerID,
		Title:       "Test Video",
		Description: "Description",
		StoragePath: "/uploads/test.mp4",
		Status:      "pending",
		Metadata:    map[string]interface{}{"size": 1024},
	}

	// 3. Настраиваем ожидания (Expectations)
	// Настраиваем ожидание для ВСЕХ 6 аргументов
	mock.ExpectQuery("INSERT INTO media_assets").
		WithArgs(
			asset.OwnerID,     // $1
			asset.Title,       // $2
			asset.Description, // $3
			asset.StoragePath, // $4
			asset.Status,      // $5
			asset.Metadata,    // $6
		).
		// Возвращаем две колонки: id и created_at (как в RETURNING)
		WillReturnRows(pgxmock.NewRows([]string{"id", "created_at"}).
			AddRow(newAssetID, now))
	// Эмулируем возврат ID, если используете RETURNING

	// 4. Вызываем метод и проверяем ассертами от stretchr
	err = repo.SaveAsset(context.Background(), asset)

	assert.NoError(t, err)                        //  проверяет, что код не вернул ошибку.
	assert.NoError(t, mock.ExpectationsWereMet()) // проверяет, что код сделал всё, что обещал сделать с базой данных.
}
