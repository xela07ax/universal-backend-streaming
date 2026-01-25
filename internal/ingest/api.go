package ingest

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

func (e *RTCEngine) HandleListStreams(sm *SessionManager, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		streams := sm.GetActiveStreams()

		w.Header().Set("Content-Type", "application/json")
		// Используем нашу стандартную структуру ответа, если она есть,
		// или просто кодируем список
		if err := json.NewEncoder(w).Encode(streams); err != nil {
			logger.Error("Failed to encode streams list", zap.Error(err))
		}
	}
}
