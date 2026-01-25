package api

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func (s *Server) respond(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	// Создаем структуру ответа
	resp := APIResponse{
		Success: code < 400,
		Data:    data,
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		s.logger.Error("failed to encode and send api response",
			zap.Int("status", code),
			zap.Error(err),
		)
	}
}

func (s *Server) respondError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	err := json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error:   message,
	})

	if err != nil {
		// Логируем ошибку, так как клиент может получить пустой или битый ответ
		s.logger.Error("failed to encode error response",
			zap.Int("code", code),
			zap.String("msg", message),
			zap.Error(err),
		)
	}
}
