package api

import (
	"net/http"
)

// APIDoc описывает структуру вашего API для фронтенда
type APIDoc struct {
	Version string            `json:"version"`
	Routes  []RouteDefinition `json:"routes"`
}

type RouteDefinition struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Description string `json:"description"`
	Protected   bool   `json:"protected"`
	// Добавляем описание параметров
	Params map[string]string `json:"params,omitempty"`
	Body   interface{}       `json:"body,omitempty"`
}

// generateDocs собирает данные о роутах без внешних утилит
func (s *Server) generateDocs() APIDoc {
	return APIDoc{
		Version: "2026.1",
		Routes: []RouteDefinition{
			{
				Method:      "POST",
				Path:        "/api/v1/login",
				Description: "Авторизация админа",
				Protected:   false,
				Body: map[string]string{
					"username": "string (required)",
					"password": "string (required)",
				},
			},
			{
				Method:      "POST",
				Path:        "/api/v1/admin/upload",
				Description: "Загрузка видеофайла",
				Protected:   true,
				Params: map[string]string{
					"title": "string (form-data field)",
					"video": "file (form-data field, max 500MB)",
				},
			},
			{
				Method:      "GET",
				Path:        "/api/v1/video/{id}",
				Description: "Получить URL стрима",
				Protected:   false,
				Params: map[string]string{
					"id": "uuid (path parameter)",
				},
			},
		},
	}
}

// handleGetDocs отдает документацию в формате JSON
func (s *Server) handleGetDocs(w http.ResponseWriter, r *http.Request) {
	s.respond(w, http.StatusOK, s.generateDocs())
}
