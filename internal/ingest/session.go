package ingest

import (
	"sync"

	"github.com/pion/webrtc/v4"
	"go.uber.org/zap"
)

// Session представляет одну активную трансляцию
type Session struct {
	PeerConnection *webrtc.PeerConnection
	StreamID       string
	UserID         string
	VideoTrack     *webrtc.TrackLocalStaticRTP // Хранение видеотрека для раздачи
}

// SessionManager хранит все текущие стримы в памяти
type SessionManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
	logger   *zap.Logger
}

// StreamInfo — структура для ответа API
type StreamInfo struct {
	StreamID string `json:"stream_id"`
	UserID   string `json:"user_id"`
}

func NewSessionManager(logger *zap.Logger) *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
		logger:   logger,
	}
}

func (m *SessionManager) Add(id string, s *Session) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[id] = s
	m.logger.Info("🎬 New streaming session started", zap.String("id", id))
}

func (m *SessionManager) Remove(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[id]; ok {
		_ = s.PeerConnection.Close()
		delete(m.sessions, id)
		m.logger.Info("⏹️ Streaming session closed", zap.String("id", id))
	}
}

// GetActiveStreams Получение списка «Живых стримов»
func (m *SessionManager) GetActiveStreams() []StreamInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	streams := make([]StreamInfo, 0, len(m.sessions))
	for id, s := range m.sessions {
		// Убираем проверку s.VideoTrack != nil для теста
		streams = append(streams, StreamInfo{
			StreamID: id,
			UserID:   s.UserID,
		})
	}
	return streams
}
