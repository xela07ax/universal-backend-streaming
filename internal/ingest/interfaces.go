package ingest

import (
	"net/http"

	"go.uber.org/zap"
)

// Streamer — возможности для тех, кто публикует контент
type Streamer interface {
	HandleWHIP(sm *SessionManager, logger *zap.Logger) http.HandlerFunc
}

// Player — возможности для тех, кто потребляет контент
type Player interface {
	HandleWHEP(sm *SessionManager, logger *zap.Logger) http.HandlerFunc
}
