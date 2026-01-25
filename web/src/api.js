import axios from 'axios';

const client = axios.create({
    baseURL: '/api/v1',
});

// Перехватчик для добавления токена
client.interceptors.request.use(config => {
    // Секрета здесь больше нет! Только временный токен пользователя.
    const token = localStorage.getItem('hydro_token');

    if (token) {
        config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
});

// Обработка ошибок (например, если токен протух)
client.interceptors.response.use(
    response => response,
    error => {
        if (error.response && error.response.status === 401) {
            localStorage.removeItem('hydro_token');
            // Здесь можно сделать редирект на страницу логина
            // window.location.href = '/login';
        }
        return Promise.reject(error);
    }
);
