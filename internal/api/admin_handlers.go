package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/xela07ax/universal-backend-streaming/internal/repository"
)

// handleAdminListAssets возвращает список всех видео для таблицы в админке
func (s *Server) handleAdminListAssets(w http.ResponseWriter, r *http.Request) {
	// В реальном проекте здесь будет s.media.ListAssets(ctx) с пагинацией
	assets, err := s.media.GetAllAssets(r.Context())
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "failed to fetch assets")
		return
	}
	s.respond(w, http.StatusOK, assets)
}

// handleAdminCreateAsset принимает данные от формы создания на фронтенде
func (s *Server) handleAdminCreateAsset(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		StoragePath string `json:"storage_path"`
		OwnerID     string `json:"owner_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ownerUUID, _ := uuid.Parse(payload.OwnerID)

	asset := &repository.MediaAsset{
		OwnerID:     ownerUUID,
		Title:       payload.Title,
		Description: payload.Description,
		StoragePath: payload.StoragePath,
		Status:      "ready",
	}

	if err := s.media.SaveAsset(r.Context(), asset); err != nil {
		s.respondError(w, http.StatusInternalServerError, "could not save asset")
		return
	}

	s.respond(w, http.StatusCreated, asset)
}
