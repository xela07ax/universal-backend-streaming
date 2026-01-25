<script setup>
import { ref, onMounted } from 'vue';
import { MediaAPI } from './utils/api';
import HydroLogin from './components/HydroLogin.vue';
import HydroPlayer from './components/HydroPlayer.vue';
import HydroManager from './components/HydroManager.vue';

const isAuthenticated = ref(false);
const assets = ref([]);
const streams = ref([]); // Живые стримы из Pion
const selectedAsset = ref(null);
const currentUrl = ref(null);

const isUploading = ref(false);
const uploadProgress = ref(0);
const managerRef = ref(null);

onMounted(() => {
  if (localStorage.getItem('hydro_token')) {
    isAuthenticated.value = true;
    refreshData();
  }
});

const refreshData = async () => {
  try {
    assets.value = await MediaAPI.getAssets();
    streams.value = await MediaAPI.getLiveStreams(); // Нужно добавить в API
  } catch (e) { isAuthenticated.value = false; }
};

const onLoginSuccess = () => {
  isAuthenticated.value = true;
  refreshData();
};

const handlePlay = async (asset) => {
  // 1. Немедленно обнуляем URL, чтобы остановить старое видео
  currentUrl.value = "";

  if (!asset) return;

  // 2. Если это LIVE, мы ВООБЩЕ не идем в API за URL
  if (asset.status === 'live') {
    // Небольшая пауза для инициализации сессии на бэкенде
    await new Promise(r => setTimeout(r, 500));
    selectedAsset.value = asset;
    return;
  }

  // 3. Только если это файл, идем в API
  try {
    const url = await MediaAPI.getVideoUrl(asset.id);
    if (url) {
      selectedAsset.value = asset;
      currentUrl.value = url;
    }
  } catch (e) {
    console.error("VOD Error", e);
  }
};

const handleUpload = async ({ file, title }) => {
  isUploading.value = true;
  try {
    await MediaAPI.uploadVideo(file, title, (p) => uploadProgress.value = p);
    managerRef.value.closeModal();
    refreshData();
  } finally { isUploading.value = false; }
};
</script>

<template>
  <div class="app-shell">
    <HydroLogin v-if="!isAuthenticated" @success="onLoginSuccess" />

    <template v-else>
      <header class="main-header">
        <h1>HYDRO <span class="blue">ENGINE</span></h1>
        <button @click="isAuthenticated = false" class="logout-btn">ВЫЙТИ</button>
      </header>

      <main class="dashboard-grid">
        <HydroManager
            ref="managerRef"
            :assets="assets"
            :streams="streams"
            :isUploading="isUploading"
            :uploadProgress="uploadProgress"
            @play="handlePlay"
            @upload="handleUpload"
            @refresh="refreshData"
        />

        <aside>
          <HydroPlayer
              v-if="selectedAsset"
              :asset="selectedAsset"
              :src="currentUrl"
              :isLive="selectedAsset.status === 'live'"
          />
        </aside>
      </main>
    </template>
  </div>
</template>

<style>
.app-shell { padding: 40px; background: #111; color: white; min-height: 100vh; }
.main-header { display: flex; justify-content: space-between; margin-bottom: 30px; }
.blue { color: #4488ff; }
.dashboard-grid { display: grid; grid-template-columns: 1fr 420px; gap: 30px; }
.logout-btn { background: #222; color: #ff4444; border: 1px solid #333; padding: 8px 16px; border-radius: 6px; cursor: pointer; }
</style>
