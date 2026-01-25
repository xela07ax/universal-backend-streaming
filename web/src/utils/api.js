import axios from 'axios';

// 1. Создаем единый экземпляр клиента
const client = axios.create({
    baseURL: '/api/v1',
    withCredentials: true // Для передачи Refresh-куки (HttpOnly)
});

// 2. Перехватчик запросов: добавляем Access Token из LocalStorage
client.interceptors.request.use(config => {
    const token = localStorage.getItem('hydro_token');
    if (token) {
        config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
});

// 3. Перехватчик ответов: логика Silent Refresh (2026 standard)
client.interceptors.response.use(
    (response) => response,
    async (error) => {
        const originalRequest = error.config;

        // Если получили 401 и это не повторный запрос
        if (error.response?.status === 401 && !originalRequest._retry) {
            originalRequest._retry = true;
            try {
                // Вызываем наш эндпоинт обновления (бэкенд проверит Refresh-куку)
                const { data } = await axios.post('/api/v1/refresh', {}, { withCredentials: true });
                const newToken = data.data.token;

                localStorage.setItem('hydro_token', newToken);

                // Повторяем упавший запрос с новым токеном
                originalRequest.headers.Authorization = `Bearer ${newToken}`;
                return client(originalRequest);
            } catch (refreshError) {
                // Если рефреш не удался (сессия в Redis удалена) — полная очистка
                localStorage.removeItem('hydro_token');
                window.location.href = '/'; // Редирект на логин
                return Promise.reject(refreshError);
            }
        }
        return Promise.reject(error);
    }
);

// 4. Группируем методы API
export const MediaAPI = {
    // Авторизация
    login: async (username, password) => {
        const { data } = await client.post('/login', { username, password });
        const token = data.data.token;
        if (token) {
            localStorage.setItem('hydro_token', token);
            return data.data; // Возвращаем весь объект (token, role, user_id)
        }
        throw new Error("Token not received");
    },

    logout: async () => {
        try { await client.post('/logout'); }
        finally {
            localStorage.removeItem('hydro_token');
            window.location.reload();
        }
    },

    // Работа с файлами (VOD)
    getAssets: () => client.get('/assets').then(res => res.data.data),

    getVideoUrl: (id) => client.get(`/video/${id}`).then(res => {
        // Логируем для отладки — в 2026 это лучший способ найти причину
        console.log('Backend response:', res.data);

        // Проверяем наличие вложенности data.url или просто url
        if (res.data && res.data.data && res.data.data.url) {
            return res.data.data.url;
        }
        if (res.data && res.data.url) {
            return res.data.url;
        }

        throw new Error("URL не найден в ответе сервера");
    }),

    uploadVideo: async (file, title, onProgress) => {
        const formData = new FormData();
        formData.append('video', file);
        formData.append('title', title);

        return client.post('/upload', formData, {
            headers: { 'Content-Type': 'multipart/form-data' },
            onUploadProgress: (p) => {
                if (onProgress) onProgress(Math.round((p.loaded * 100) / p.total));
            }
        }).then(res => res.data.data);
    },


    // --- НОВОЕ: WebRTC Стриминг (ветка feature/webrtc-streaming) ---

    // Получить список активных трансляций из SessionManager
    getLiveStreams: () => client.get('/streams').then(res => {
        const data = res.data.data || res.data;
        console.log("📥 Raw Streams from Server:", data); // Посмотри это в консоли!
        return data;
    }),
    // Метод для инициализации WHEP (зритель)
    // Мы не используем axios для SDP, так как fetch удобнее для текстовых потоков WebRTC
    getWhepAnswer: async (streamId, offerSdp) => {
        const response = await fetch(`/api/v1/whep?stream_id=${streamId}`, {
            method: 'POST',
            body: offerSdp,
            headers: {
                'Content-Type': 'application/sdp',
                'Authorization': `Bearer ${localStorage.getItem('hydro_token')}`
            }
        });
        if (!response.ok) throw new Error('WHEP handshake failed');
        return await response.text();
    }
};
