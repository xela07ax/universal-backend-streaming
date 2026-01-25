<template>
  <div style="padding: 40px; font-family: 'Inter', sans-serif; background: #111; color: white; min-height: 100vh;">

    <!-- 1. ЭКРАН АВТОРИЗАЦИИ -->
    <div v-if="!isAuthenticated" style="position: fixed; inset: 0; background: #0a0a0a; display: flex; align-items: center; justify-content: center; z-index: 9999;">
      <div style="background: #1a1a1a; padding: 40px; border-radius: 16px; width: 360px; border: 1px solid #333; box-shadow: 0 20px 40px rgba(0,0,0,0.6);">
        <h2 style="margin-top: 0; color: #4488ff; letter-spacing: 2px; text-align: center;">HYDRO <span style="color:white">LOGIN</span></h2>
        <div style="margin-top: 30px;">
          <input v-model="loginForm.username" type="text" placeholder="Логин" style="width: 100%; padding: 12px; margin-bottom: 15px; background: #000; border: 1px solid #333; color: white; border-radius: 8px; box-sizing: border-box;" />
          <input v-model="loginForm.password" type="password" placeholder="Пароль" @keyup.enter="handleLogin" style="width: 100%; padding: 12px; margin-bottom: 25px; background: #000; border: 1px solid #333; color: white; border-radius: 8px; box-sizing: border-box;" />
          <button @click="handleLogin" style="width: 100%; background: #4488ff; color: white; border: none; padding: 14px; border-radius: 8px; cursor: pointer; font-weight: bold; transition: 0.2s;">ВОЙТИ</button>
          <p v-if="authError" style="color: #ff4444; font-size: 13px; margin-top: 15px; text-align: center;">{{ authError }}</p>
        </div>
      </div>
    </div>

    <!-- 2. ОСНОВНОЙ ИНТЕРФЕЙС АДМИНКИ -->
    <template v-else>
      <header style="display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid #222; padding-bottom: 20px; margin-bottom: 30px;">
        <div>
          <h1 style="margin: 0; font-size: 24px;">HYDRO <span style="color: #4488ff;">ENGINE</span></h1>

          <div style="font-size: 11px; color: #555; margin-top: 4px;">PROJECT: universal-backend-streaming</div>
        </div>
        <div style="display: flex; gap: 15px;">
          <button @click="showModal = true" style="background: #4488ff; color: white; border: none; padding: 10px 20px; border-radius: 6px; cursor: pointer; font-weight: bold;">+ ДОБАВИТЬ ВИДЕО</button>
          <div style="display: flex; gap: 20px; align-items: center;">
            <span style="color: #666; font-size: 13px;">Admin Mode</span>
            <button @click="handleLogout" style="background: #333; color: #ff4444; border: 1px solid #444; padding: 8px 16px; border-radius: 6px; cursor: pointer; font-weight: bold;">
              ВЫЙТИ
            </button>
          </div>
        </div>
      </header>

      <div style="display: grid; grid-template-columns: 1fr 420px; gap: 30px;">
        <!-- Списочная часть -->
        <section style="background: #161616; border-radius: 12px; border: 1px solid #222; overflow: hidden;">
          <table style="width: 100%; border-collapse: collapse; text-align: left;">
            <thead style="background: #1a1a1a; color: #555; font-size: 11px; text-transform: uppercase;">
            <tr>
              <th style="padding: 15px;">Медиафайл</th>
              <th style="padding: 15px;">Статус</th>
              <th style="padding: 15px; text-align: right;">Действие</th>
            </tr>
            </thead>
            <tbody>
            <!-- ИСПОЛЬЗУЕМ МАЛЕНЬКИЙ РЕГИСТР: asset.id, asset.title, asset.status -->
            <tr v-for="asset in assets" :key="asset.id" style="border-bottom: 1px solid #222;">
              <td style="padding: 15px;">
                <div style="font-weight: 500;">{{ asset.title }}</div>
                <div style="font-size: 11px; color: #444;">{{ asset.storage_path }}</div>
              </td>
              <td style="padding: 15px;">
                <span style="color: #44ff88; font-size: 12px; background: rgba(68,255,136,0.1); padding: 4px 8px; border-radius: 4px;">{{ asset.status }}</span>
              </td>
              <td style="padding: 15px; text-align: right;">
                <button @click="playVideo(asset)" style="color: #4488ff; background: none; border: 1px solid #222; padding: 6px 12px; border-radius: 4px; cursor: pointer;">Смотреть</button>
              </td>
            </tr>
            <tr v-if="assets && assets.length === 0">
              <td colspan="3" style="padding: 40px; text-align: center; color: #444;">Данные не найдены</td>
            </tr>
            </tbody>
          </table>
        </section>

        <!-- Плеер -->
        <aside>
          <div v-if="currentUrl" style="background: #000; padding: 20px; border-radius: 12px; border: 1px solid #222; position: sticky; top: 20px;">
            <video :key="currentUrl" :src="currentUrl" controls autoplay style="width: 100%; border-radius: 8px; background: #000;"></video>
            <div style="margin-top: 15px;">
              <h3 style="margin: 0; font-size: 18px;">{{ selectedAsset?.title }}</h3>
              <p style="color: #666; font-size: 12px; margin-top: 10px; word-break: break-all;">Source: {{ currentUrl }}</p>
            </div>
          </div>
          <div v-else style="height: 300px; border: 2px dashed #222; border-radius: 12px; display: flex; align-items: center; justify-content: center; color: #333;">
            Выберите видео для просмотра
          </div>
        </aside>
      </div>

      <!-- МОДАЛЬНОЕ ОКНО ЗАГРУЗКИ -->
      <div v-if="showModal" style="position: fixed; inset: 0; background: rgba(0,0,0,0.85); display: flex; align-items: center; justify-content: center; z-index: 3000;">
        <div style="background: #1a1a1a; padding: 35px; border-radius: 16px; width: 420px; border: 1px solid #333;">
          <h3 style="margin-top: 0; margin-bottom: 25px;">Загрузка контента</h3>

          <div style="margin-bottom: 20px;">
            <label style="display: block; font-size: 11px; color: #555; margin-bottom: 8px;">Название видео</label>
            <input v-model="newVideoTitle" type="text" style="width: 100%; padding: 12px; background: #0a0a0a; border: 1px solid #333; color: white; border-radius: 8px; box-sizing: border-box;" />
          </div>

          <div style="margin-bottom: 25px; border: 2px dashed #333; padding: 20px; text-align: center; border-radius: 8px; background: #0f0f0f;">
            <input type="file" @change="onFileSelected" accept="video/*" id="file-upload" style="display: none;" />
            <label for="file-upload" style="cursor: pointer; color: #4488ff; font-size: 14px;">
              {{ selectedFile ? selectedFile.name : 'Выбрать файл' }}
            </label>
          </div>

          <!-- Прогресс бар -->
          <div v-if="isUploading" style="margin-bottom: 20px;">
            <div style="height: 4px; background: #333; width: 100%; border-radius: 2px; overflow: hidden;">
              <div :style="{ width: uploadProgress + '%' }" style="height: 100%; background: #4488ff; transition: width 0.3s ease;"></div>
            </div>
            <div style="text-align: right; font-size: 10px; color: #4488ff; margin-top: 5px;">{{ uploadProgress }}%</div>
          </div>

          <div style="display: flex; gap: 15px; justify-content: flex-end;">
            <button @click="closeModal" :disabled="isUploading" style="background: transparent; color: #555; border: none; cursor: pointer;">ОТМЕНА</button>
            <button @click="submitUpload" :disabled="isUploading" style="background: #4488ff; color: white; border: none; padding: 10px 25px; border-radius: 6px; cursor: pointer; font-weight: bold;">
              {{ isUploading ? 'ЗАГРУЗКА...' : 'ОТПРАВИТЬ' }}
            </button>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { MediaAPI } from './utils/api';

const isAuthenticated = ref(false);
const authError = ref('');
const loginForm = ref({ username: 'admin', password: '' });

const assets = ref([]);
const selectedAsset = ref(null);
const currentUrl = ref(null);

const showModal = ref(false);
const isUploading = ref(false);
const uploadProgress = ref(0);
const newVideoTitle = ref('');
const selectedFile = ref(null);

onMounted(() => {
  const token = localStorage.getItem('hydro_token');
  if (token) {
    isAuthenticated.value = true;
    loadAssets();
  } else {
    // Если токена нет — сразу в логаут
    isAuthenticated.value = false;
    assets.value = [];
  }
});

const handleLogin = async () => {
  try {
    await MediaAPI.login(loginForm.value.username, loginForm.value.password);
    isAuthenticated.value = true;
    authError.value = '';
    loadAssets();
  } catch (err) {
    authError.value = "Неверный пароль или ошибка сервера";
  }
};

const handleLogout = async () => {
  if (confirm("Вы уверены, что хотите выйти?")) {
    await MediaAPI.logout();
    isAuthenticated.value = false;
  }
};
const loadAssets = async () => {
  try {
    const data = await MediaAPI.getAssets();
    assets.value = data;
  } catch (err) {
    // Если запрос упал с 401, и интерцептор не смог его починить
    if (err.response?.status === 401) {
      assets.value = []; // ГАРАНТИРУЕМ очистку списка
      isAuthenticated.value = false;
    }
  }
};

const playVideo = async (asset) => {
  selectedAsset.value = asset;
  try {
    currentUrl.value = await MediaAPI.getVideoUrl(asset.id);
  } catch (e) {
    alert("Ошибка получения видео");
  }
};

const onFileSelected = (e) => {
  selectedFile.value = e.target.files[0];
};

const submitUpload = async () => {
  if (!selectedFile.value || !newVideoTitle.value) {
    alert("Заполните поля");
    return;
  }
  isUploading.value = true;
  uploadProgress.value = 0;
  try {
    await MediaAPI.uploadVideo(
        selectedFile.value,
        newVideoTitle.value,
        (p) => { uploadProgress.value = p; }
    );
    closeModal();
    await loadAssets();
    alert("Загружено!");
  } catch (err) {
    alert("Ошибка загрузки");
  } finally {
    isUploading.value = false;
  }
};

const closeModal = () => {
  showModal.value = false;
  newVideoTitle.value = '';
  selectedFile.value = null;
};
</script>
