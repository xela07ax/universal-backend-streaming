package ingest

import (
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pion/webrtc/v4"
	"go.uber.org/zap"
)

func (e *RTCEngine) HandleWHEP(sm *SessionManager, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Получаем ID стрима (пробуем Query и URL Param для гибкости)
		streamID := r.URL.Query().Get("stream_id")
		if streamID == "" {
			streamID = chi.URLParam(r, "id")
		}

		sm.mu.RLock()
		session, ok := sm.sessions[streamID]
		sm.mu.RUnlock()
		logger.Info("🔍 WHEP: Searching for stream", zap.String("requested_id", streamID))
		if !ok || session.VideoTrack == nil {
			logger.Warn("WHEP: Stream not found or no track", zap.String("id", streamID))
			http.Error(w, "Stream not found or not ready", http.StatusNotFound)
			return
		}

		// 2. Читаем Offer SDP от плеера
		offerSDP, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Invalid SDP", http.StatusBadRequest)
			return
		}

		// 3. Создаем PeerConnection для зрителя
		// В 2026 году для локалки используем пустую конфигурацию
		pc, err := e.api.NewPeerConnection(webrtc.Configuration{})
		if err != nil {
			logger.Error("WHEP: PC creation failed", zap.Error(err))
			return
		}

		// 4. ДОБАВЛЯЕМ ТРЕК СТРИМЕРА ЗРИТЕЛЮ
		rtpSender, err := pc.AddTrack(session.VideoTrack)
		if err != nil {
			logger.Error("WHEP: Failed to add track", zap.Error(err))
			return
		}

		// Читаем RTCP (важно для работы обратной связи по качеству)
		go func() {
			buf := make([]byte, 1500)
			for {
				if _, _, err := rtpSender.Read(buf); err != nil {
					return
				}
			}
		}()

		// 5. Устанавливаем Remote Description
		err = pc.SetRemoteDescription(webrtc.SessionDescription{
			Type: webrtc.SDPTypeOffer,
			SDP:  string(offerSDP),
		})
		if err != nil {
			logger.Error("WHEP: SetRemote err", zap.Error(err))
			return
		}

		// 6. Создаем Answer
		answer, err := pc.CreateAnswer(nil)
		if err != nil {
			logger.Error("WHEP: CreateAnswer err", zap.Error(err))
			return
		}

		// 7. ЖДЕМ СБОРА ICE-КАНДИДАТОВ (Важно для 2026!)
		// Если отправить ответ сразу, зритель может не найти путь к серверу
		gatherFinished := webrtc.GatheringCompletePromise(pc)

		err = pc.SetLocalDescription(answer)
		if err != nil {
			return
		}

		<-gatherFinished // Ждем завершения сбора

		// 8. Отдаем финальный Answer со всеми кандидатами
		w.Header().Set("Content-Type", "application/sdp")
		w.Header().Set("Access-Control-Allow-Origin", "*") // Для работы плеера
		w.WriteHeader(http.StatusCreated)

		// Отправляем текущий LocalDescription (он уже содержит кандидатов)
		_, _ = w.Write([]byte(pc.LocalDescription().SDP))

		logger.Info("✅ WHEP: Viewer connected", zap.String("stream_id", streamID))
	}
}
