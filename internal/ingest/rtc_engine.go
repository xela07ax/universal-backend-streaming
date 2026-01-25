package ingest

import (
	"net"

	"github.com/pion/ice/v4"
	"github.com/pion/webrtc/v4"
	"go.uber.org/zap"
)

type RTCEngine struct {
	api *webrtc.API
}

func NewRTCEngine(logger *zap.Logger) (*RTCEngine, error) {
	m := &webrtc.MediaEngine{}
	if err := m.RegisterDefaultCodecs(); err != nil {
		return nil, err
	}

	s := webrtc.SettingEngine{}
	// Если ты на Windows, это заставит Pion предлагать локальный адрес
	//s.SetNAT1To1IPs([]string{"127.0.0.1"}, webrtc.ICECandidateTypeHost)
	// 1. Настройка UDP Mux
	// Используем IPv4zero (0.0.0.0), чтобы Pion мог слушать на всех интерфейсах
	udpAddr := &net.UDPAddr{IP: net.IPv4zero, Port: 50000}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		logger.Warn("⚠️ RTC: UDP bind failed", zap.Error(err))
	} else {
		udpMux := ice.NewUDPMuxDefault(ice.UDPMuxParams{
			UDPConn: udpConn,
		})
		s.SetICEUDPMux(udpMux)
		logger.Info("📡 RTC: UDP Mux active", zap.Int("port", 50000))
	}

	// 2. Настройка TCP Mux (Помогает пробиться через строгие брандмауэры)
	tcpAddr := &net.TCPAddr{IP: net.IPv4zero, Port: 3478}
	tcpListener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		logger.Warn("⚠️ RTC: TCP bind failed (non-critical)", zap.Error(err))
	} else {
		tcpMux := ice.NewTCPMuxDefault(ice.TCPMuxParams{
			Listener: tcpListener,
		})
		s.SetICETCPMux(tcpMux)
		logger.Info("🌐 RTC: TCP ICE Listener active", zap.Int("port", 3478))
	}

	// ВАЖНО: Для локальной разработки на Windows 2026 НЕ ИСПОЛЬЗУЙТЕ:
	// - s.SetLite(true) -> ломает ICE на localhost
	// - s.SetNAT1To1IPs -> часто вызывает "invalid address rewrite"

	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(m),
		webrtc.WithSettingEngine(s),
	)

	return &RTCEngine{api: api}, nil
}
