package streaming

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestVideoProvider_GetSafePath(t *testing.T) {
	// Создаем провайдер с базовым путем в папке проекта
	basePath := "./storage/uploads"
	p := &VideoProvider{
		basePath: basePath,
		logger:   zap.NewNop(),
	}

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "Valid file",
			input:   "video.mp4",
			wantErr: false,
		},
		{
			name:    "Subdirectory file",
			input:   "users/1/clip.mp4",
			wantErr: false,
		},
		{
			name:    "Path Traversal Attempt",
			input:   "../../../etc/passwd",
			wantErr: true, // Должен выдать ошибку безопасности
		},
		{
			name:    "Hidden config access",
			input:   "../../configs/hydro.yaml",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := p.GetSafePath(tt.input) // Используем наш метод защиты
			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, path)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, path, "uploads")
			}
		})
	}
}
