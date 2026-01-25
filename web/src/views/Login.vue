<script setup>
import { ref } from 'vue';
import axios from 'axios';

const username = ref('');
const password = ref('');
const error = ref('');

const login = async () => {
  try {
    const res = await axios.post('/api/v1/login', {
      username: username.value,
      password: password.value
    });

    // Сохраняем настоящий JWT
    localStorage.setItem('hydro_token', res.data.data.token);
    window.location.reload(); // Перезагрузка для применения токена в axios
  } catch (err) {
    error.value = 'Неверный логин или пароль';
  }
};
</script>

<template>
  <div class="login-box">
    <h2>HYDRO LOGIN</h2>
    <input v-model="username" placeholder="Логин" />
    <input v-model="password" type="password" placeholder="Пароль" />
    <button @click="login">Войти</button>
    <p v-if="error" class="error">{{ error }}</p>
  </div>
</template>
