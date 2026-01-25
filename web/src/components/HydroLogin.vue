<template>
  <div class="auth-overlay">
    <div class="auth-card">
      <div class="auth-header">
        <h2>HYDRO <span class="highlight">LOGIN</span></h2>
        <p class="project-tag">v2026.1.1 Core</p>
      </div>

      <div class="auth-form">
        <div class="input-group">
          <input
              v-model="form.username"
              type="text"
              placeholder="Логин"
              class="hydro-input"
          />
        </div>
        <div class="input-group">
          <input
              v-model="form.password"
              type="password"
              placeholder="Пароль"
              @keyup.enter="onSubmit"
              class="hydro-input"
          />
        </div>

        <button
            @click="onSubmit"
            :disabled="loading"
            class="hydro-button primary"
        >
          {{ loading ? 'АВТОРИЗАЦИЯ...' : 'ВОЙТИ' }}
        </button>

        <p v-if="error" class="error-msg">{{ error }}</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue';
import { MediaAPI } from '../utils/api';

const emit = defineEmits(['success']);

const loading = ref(false);
const error = ref('');
const form = reactive({
  username: 'admin',
  password: ''
});

const onSubmit = async () => {
  if (!form.username || !form.password) {
    error.value = "Заполните все поля";
    return;
  }

  loading.value = true;
  error.value = '';

  try {
    const data = await MediaAPI.login(form.username, form.password);
    // Бэкенд возвращает токен, мы передаем успех родителю
    emit('success', data);
  } catch (err) {
    error.value = err.response?.data?.error || "Неверный пароль или ошибка сервера";
  } finally {
    loading.value = false;
  }
};
</script>

<style scoped>
.auth-overlay {
  position: fixed;
  inset: 0;
  background: #0a0a0a;
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
}

.auth-card {
  background: #1a1a1a;
  padding: 40px;
  border-radius: 16px;
  width: 360px;
  border: 1px solid #333;
  box-shadow: 0 20px 40px rgba(0,0,0,0.6);
}

.auth-header h2 { margin: 0; color: white; letter-spacing: 2px; text-align: center; }
.highlight { color: #4488ff; }
.project-tag { font-size: 10px; color: #444; text-align: center; margin-top: 5px; }

.auth-form { margin-top: 30px; }

.hydro-input {
  width: 100%;
  padding: 12px;
  margin-bottom: 15px;
  background: #000;
  border: 1px solid #333;
  color: white;
  border-radius: 8px;
  box-sizing: border-box;
  outline: none;
}

.hydro-input:focus { border-color: #4488ff; }

.hydro-button {
  width: 100%;
  background: #4488ff;
  color: white;
  border: none;
  padding: 14px;
  border-radius: 8px;
  cursor: pointer;
  font-weight: bold;
}

.hydro-button:disabled { opacity: 0.5; }

.error-msg {
  color: #ff4444;
  font-size: 13px;
  margin-top: 15px;
  text-align: center;
}
</style>
