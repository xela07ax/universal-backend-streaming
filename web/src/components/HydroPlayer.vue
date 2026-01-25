<template>
  <div class="hydro-player-card">
    <div class="video-container">
      <video
          ref="videoRef"
          controls
          autoplay
          playsinline
          crossorigin="anonymous"
          class="main-video"
      ></video>
      <div v-if="isLive" class="live-indicator">
        <span class="dot"></span> LIVE
      </div>
    </div>

    <div v-if="asset" class="video-info">
      <h3>{{ asset.title }}</h3>
      <div class="meta">
        <span class="badge">{{ isLive ? 'WebRTC Stream' : 'VOD Asset' }}</span>
        <span class="id">ID: {{ asset.id }}</span>
      </div>
      <p class="source-text">Источник: {{ isLive ? 'Ingest Engine' : src }}</p>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, onUnmounted } from 'vue';

const props = defineProps({
  asset: Object,
  src: String,
  isLive: Boolean
});

const videoRef = ref(null);
let pc = null;

// 1. Сначала определяем вспомогательные функции
const stopWebRTC = () => {
  if (pc) {
    pc.close();
    pc = null;
  }
  if (videoRef.value) {
    videoRef.value.srcObject = null;
    videoRef.value.src = '';
  }
};

const startWhep = async (streamId) => {
  stopWebRTC(); // Чистим старое соединение

  pc = new RTCPeerConnection({
    iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
  });

  pc.ontrack = (event) => {
    console.log("📡 Получен медиа-трек от сервера");
    if (videoRef.value && event.streams && event.streams[0]) {
      videoRef.value.srcObject = event.streams[0];
    }
  };

  try {
    const offer = await pc.createOffer({ offerToReceiveVideo: true });
    await pc.setLocalDescription(offer);

    const token = localStorage.getItem('hydro_token');
    const response = await fetch(`/api/v1/whep?stream_id=${streamId}`, {
      method: 'POST',
      body: offer.sdp,
      headers: {
        'Content-Type': 'application/sdp',
        'Authorization': `Bearer ${token}`
      }
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(`WHEP Error: ${response.status} - ${errorText}`);
    }

    const answerSdp = await response.text();

    if (pc && pc.signalingState !== 'closed') {
      await pc.setRemoteDescription({ type: 'answer', sdp: answerSdp });
      console.log("✅ WebRTC сессия установлена!");
    }
  } catch (err) {
    console.error("❌ Ошибка WHEP:", err);
  }
};

// 2. И только в самом конце вешаем watch, который будет их использовать
watch(() => props.asset, (newAsset) => {
  if (!newAsset) return;

  if (props.isLive) {
    console.log("🚀 Инициализация WebRTC для стрима:", newAsset.id);
    startWhep(newAsset.id);
  } else {
    stopWebRTC();
    if (videoRef.value && props.src) {
      videoRef.value.src = props.src;
    }
  }
}, { immediate: true });

onUnmounted(() => stopWebRTC());
</script>


<style scoped>
.hydro-player-card {
  background: #000;
  padding: 20px;
  border-radius: 12px;
  border: 1px solid #222;
  position: sticky;
  top: 20px;
}
.video-container { position: relative; }
.main-video {
  width: 100%;
  border-radius: 8px;
  background: #050505;
  box-shadow: 0 4px 20px rgba(0,0,0,0.5);
  aspect-ratio: 16 / 9;
}
.live-indicator {
  position: absolute;
  top: 15px;
  left: 15px;
  background: rgba(255, 68, 68, 0.9);
  color: white;
  padding: 4px 10px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: bold;
  display: flex;
  align-items: center;
  gap: 6px;
}
.dot {
  width: 8px;
  height: 8px;
  background: white;
  border-radius: 50%;
  animation: blink 1s infinite;
}
.video-info { margin-top: 20px; }
.video-info h3 { margin: 0; font-size: 20px; color: #eee; }
.meta { display: flex; gap: 10px; align-items: center; margin-top: 10px; }
.badge { background: #4488ff22; color: #4488ff; font-size: 10px; padding: 2px 8px; border-radius: 4px; text-transform: uppercase; }
.id { font-size: 11px; color: #444; }
.source-text { color: #666; font-size: 12px; margin-top: 15px; word-break: break-all; }
@keyframes blink { 0% { opacity: 1; } 50% { opacity: 0.4; } 100% { opacity: 1; } }
</style>
