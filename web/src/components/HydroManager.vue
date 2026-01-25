<script setup>
import { ref } from 'vue';

const props = defineProps({
  assets: Array,
  streams: Array,
  isUploading: Boolean,
  uploadProgress: Number
});

const emit = defineEmits(['play', 'upload', 'refresh']);

const showModal = ref(false);
const newVideoTitle = ref('');
const selectedFile = ref(null);

// Исправлено: берем первый файл из массива
const onFileSelected = (e) => {
  if (e.target.files.length > 0) {
    selectedFile.value = e.target.files[0];
  }
};

const submitUpload = () => {
  if (!selectedFile.value || !newVideoTitle.value) {
    return alert("Пожалуйста, укажите название и выберите файл");
  }
  emit('upload', { file: selectedFile.value, title: newVideoTitle.value });
};

const openModal = () => { showModal.value = true; };
const closeModal = () => {
  showModal.value = false;
  newVideoTitle.value = '';
  selectedFile.value = null;
};

defineExpose({ closeModal });
</script>

<template>
  <section class="manager-container">
    <div class="actions-bar">
      <button @click="openModal" class="btn-primary">+ ДОБАВИТЬ ВИДЕО</button>
      <button @click="emit('refresh')" class="btn-secondary">🔄 ОБНОВИТЬ</button>
    </div>

    <div class="table-wrapper">
      <table class="hydro-table">
        <thead>
        <tr>
          <th>Медиафайл</th>
          <th>Статус</th>
          <th class="text-right">Действие</th>
        </tr>
        </thead>
        <tbody>
        <!-- 1. ЖИВЫЕ СТРИМЫ (Используем stream_id) -->
        <tr v-for="stream in streams" :key="stream.stream_id" class="live-row">
          <td>
            <div class="title">LIVE: {{ stream.user_id }}</div>
            <div class="sub">WebRTC Stream Active</div>
          </td>
          <td><span class="status-badge live">LIVE</span></td>
          <td class="text-right">
            <!-- ВАЖНО: передаем stream.stream_id в поле id -->
            <button
                @click="emit('play', { id: stream.stream_id, title: 'Live: ' + stream.user_id, status: 'live' })"
                class="btn-play"
            >
              Смотреть
            </button>
          </td>
        </tr>

        <!-- 2. СТАТИЧНЫЕ ФАЙЛЫ -->
        <tr v-for="asset in assets" :key="asset.id">
          <td>
            <div class="title">{{ asset.title }}</div>
            <div class="sub">{{ asset.storage_path }}</div>
          </td>
          <td><span class="status-badge">{{ asset.status }}</span></td>
          <td class="text-right">
            <button @click="emit('play', asset)" class="btn-play">Смотреть</button>
          </td>
        </tr>
        </tbody>
      </table>
    </div>

    <!-- МОДАЛКА ЗАГРУЗКИ -->
    <div v-if="showModal" class="modal-overlay">
      <div class="modal-card">
        <h3>Загрузка контента</h3>
        <input v-model="newVideoTitle" type="text" placeholder="Название видео" class="hydro-input" />

        <div class="file-dropzone">
          <input type="file" @change="onFileSelected" id="file-up" hidden />
          <label for="file-up" class="file-label">
            {{ selectedFile ? selectedFile.name : 'Выбрать MP4 файл' }}
          </label>
        </div>

        <div v-if="isUploading" class="progress-container">
          <div class="progress-bar" :style="{ width: uploadProgress + '%' }"></div>
          <div class="progress-text">{{ uploadProgress }}%</div>
        </div>

        <div class="modal-actions">
          <button @click="closeModal" :disabled="isUploading" class="btn-cancel">ОТМЕНА</button>
          <button @click="submitUpload" :disabled="isUploading" class="btn-primary">
            {{ isUploading ? 'ЗАГРУЗКА...' : 'ОТПРАВИТЬ' }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
/* Сохраняем твои стили, добавляем небольшие исправления для UX */
.manager-container { background: #161616; border-radius: 12px; border: 1px solid #222; }
.actions-bar { padding: 20px; display: flex; gap: 10px; border-bottom: 1px solid #222; }
.hydro-table { width: 100%; border-collapse: collapse; }
.hydro-table th { padding: 15px; color: #555; font-size: 11px; text-transform: uppercase; text-align: left; }
.hydro-table td { padding: 15px; border-bottom: 1px solid #222; }
.live-row { background: rgba(255, 68, 68, 0.05); }
.status-badge { font-size: 11px; background: #222; padding: 4px 8px; border-radius: 4px; color: #888; }
.status-badge.live { background: rgba(255, 68, 68, 0.2); color: #ff4444; font-weight: bold; }
.btn-primary { background: #4488ff; color: white; border: none; padding: 10px 20px; border-radius: 6px; cursor: pointer; font-weight: bold; }
.btn-primary:disabled { opacity: 0.5; }
.btn-secondary { background: #222; color: #eee; border: 1px solid #333; padding: 10px 15px; border-radius: 6px; cursor: pointer; }
.btn-play { color: #4488ff; background: none; border: 1px solid #222; padding: 6px 12px; border-radius: 4px; cursor: pointer; }
.btn-cancel { background: transparent; color: #555; border: none; cursor: pointer; padding: 10px; }
.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.8); display: flex; align-items: center; justify-content: center; z-index: 100; }
.modal-card { background: #1a1a1a; padding: 30px; border-radius: 12px; width: 400px; border: 1px solid #333; }
.hydro-input { width: 100%; padding: 12px; background: #0a0a0a; border: 1px solid #333; color: white; border-radius: 8px; margin-bottom: 20px; box-sizing: border-box; }
.progress-container { margin: 20px 0; }
.progress-bar { height: 4px; background: #4488ff; transition: width 0.3s; border-radius: 2px; }
.progress-text { text-align: right; font-size: 10px; color: #4488ff; margin-top: 5px; }
.file-dropzone { border: 2px dashed #333; padding: 20px; text-align: center; margin-bottom: 20px; border-radius: 8px; background: #0f0f0f; }
.file-label { cursor: pointer; color: #4488ff; font-size: 14px; }
.modal-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 20px; }
.text-right { text-align: right; }
</style>
