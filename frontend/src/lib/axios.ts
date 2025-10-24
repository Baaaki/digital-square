import axios from 'axios'

// Backend API'ye istek atmak için axios instance
export const api = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api',
  withCredentials: true, // ✅ Cookie'leri otomatik gönder/al
})

// ❌ Token interceptor'ı kaldırdık (cookie otomatik gönderilir)

export default api