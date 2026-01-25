import axios from 'axios';

// Создаем экземпляр
const client = axios.create({
    baseURL: '/api/v1',
    withCredentials: true // ОБЯЗАТЕЛЬНО для передачи Refresh-куки!
});

// Перехватчик запросов (добавляет Access Token)
client.interceptors.request.use(config => {
    const token = localStorage.getItem('hydro_token');
    if (token) {
        config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
});

// Перехватчик ответов (логика Silent Refresh)
client.interceptors.response.use(
    (response) => response,
    async (error) => {
        const originalRequest = error.config;

        // Если 401 и мы еще не пробовали рефреш
        if (error.response?.status === 401 && !originalRequest._retry) {
            originalRequest._retry = true;
            try {
                // Пытаемся получить новый токен
                const { data } = await axios.post('/api/v1/refresh', {}, { withCredentials: true });
                const newToken = data.data.token;

                localStorage.setItem('hydro_token', newToken);

                // Повторяем изначальный запрос с новым токеном
                originalRequest.headers.Authorization = `Bearer ${newToken}`;
                return client(originalRequest);
            } catch (refreshError) {
                // Если даже рефреш сдох — чистим всё и на выход
                localStorage.removeItem('hydro_token');
                window.location.reload();
                return Promise.reject(refreshError);
            }
        }
        return Promise.reject(error);
    }
);

export const MediaAPI = {
    // ВАЖНО: Используйте именно client, а не глобальный axios!
    getAssets: () => client.get('/admin/assets').then(res => res.data.data),

    login: async (username, password) => {
        // Делаем запрос к нашему Go-серверу
        const { data } = await client.post('/login', { username, password });

        // В нашем бэкенде ответ обернут в { success: true, data: { token: "..." } }
        const token = data.data.token;

        if (token) {
            localStorage.setItem('hydro_token', token);
            console.log('Токен успешно сохранен в базу браузера');
            return token;
        }
        throw new Error("Токен не получен");
    },
    logout: async () => {
        try {
            // Опционально уведомляем бэкенд
            await client.post('/logout');
        } finally {
            // В ЛЮБОМ СЛУЧАЕ удаляем токен из браузера
            localStorage.removeItem('hydro_token');
            // Перезагружаем страницу или редиректим на логин
            window.location.reload();
        }
    },

    getVideoUrl: (id) => client.get(`/video/${id}`).then(res => res.data.data.url),

    uploadVideo: async (file, title) => {
        const formData = new FormData();
        formData.append('video', file);
        formData.append('title', title);

        const { data } = await client.post('/admin/upload', formData, {
            headers: { 'Content-Type': 'multipart/form-data' }
        });
        return data.data;
    }
};
