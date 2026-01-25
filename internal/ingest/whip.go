package ingest

import (
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/pion/webrtc/v4"
	"github.com/xela07ax/universal-backend-streaming/internal/types"
	"go.uber.org/zap"
)

func (e *RTCEngine) HandleWHIP(sm *SessionManager, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Извлекаем UserID
		val := r.Context().Value(types.UserIDKey)
		uid, ok := val.(uuid.UUID)
		if !ok {
			logger.Error("WHIP: UserID not found in context", zap.Any("raw_val", val))
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// 2. Читаем Offer SDP
		offerSDP, err := io.ReadAll(r.Body)
		if err != nil {
			logger.Error("WHIP: Read body error", zap.Error(err))
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// 3. ПРЕДВАРИТЕЛЬНАЯ ИНИЦИАЛИЗАЦИЯ ТРЕКА (Решение 404 ошибки)
		// Создаем дефолтные параметры для H264 (стандарт OBS)
		// Это позволяет WHEP-зрителю подключиться мгновенно
		capability := webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264}
		localTrack, err := webrtc.NewTrackLocalStaticRTP(capability, "video", "hydro-stream")
		if err != nil {
			logger.Error("WHIP: Failed to pre-create local track", zap.Error(err))
			return
		}

		// 4. Создаем сессию и СРАЗУ кладем туда трек
		streamID := uuid.New().String()
		currentSession := &Session{
			StreamID:   streamID,
			UserID:     uid.String(),
			VideoTrack: localTrack, // Теперь трек доступен СРАЗУ
		}

		// 5. Создаем PeerConnection
		pc, err := e.api.NewPeerConnection(webrtc.Configuration{})
		if err != nil {
			logger.Error("WHIP: PC creation failed", zap.Error(err))
			return
		}
		currentSession.PeerConnection = pc

		// 6. Обработка входящего потока (Fan-out)
		pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
			logger.Info("📡 Ingest: Media flow started",
				zap.String("id", streamID),
				zap.String("kind", track.Kind().String()))

			// Пересылаем пакеты из входящего трека в наш заранее созданный localTrack
			for {
				packet, _, err := track.ReadRTP()
				if err != nil {
					logger.Warn("⏹️ Ingest: Track closed", zap.String("id", streamID))
					return
				}
				// Пишем пакеты в локальный трек, который уже смотрят зрители
				if err := localTrack.WriteRTP(packet); err != nil {
					return
				}
			}
		})

		// 7. Мониторинг состояния
		pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
			logger.Info("📶 RTC State Change", zap.String("id", streamID), zap.String("state", state.String()))
			// При failed/closed сессия удалится из списка активных
			if state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateClosed {
				sm.Remove(streamID)
			}
		})

		// 8. SDP Handshake
		if err := pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: string(offerSDP)}); err != nil {
			logger.Error("WHIP: SetRemoteDescription failed", zap.Error(err))
			return
		}

		answer, err := pc.CreateAnswer(nil)
		if err != nil {
			logger.Error("WHIP: CreateAnswer failed", zap.Error(err))
			return
		}

		if err := pc.SetLocalDescription(answer); err != nil {
			logger.Error("WHIP: SetLocalDescription failed", zap.Error(err))
			return
		}

		// 9. Финальная регистрация сессии
		sm.Add(streamID, currentSession)

		// 10. Ответ OBS по стандарту RFC
		w.Header().Set("Content-Type", "application/sdp")
		w.Header().Set("Location", "/api/v1/ingest/whip/"+streamID)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(answer.SDP))

		logger.Debug("🚀 WHIP Session Initialized", zap.String("id", streamID))
	}
}
