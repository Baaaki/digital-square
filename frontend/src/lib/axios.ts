import axios from 'axios'

// Backend API URL - use environment variable or fallback to Docker host port
const getBaseURL = () => {
  // In browser, check if env var exists (Next.js injects it at build time)
  if (typeof window !== 'undefined') {
    // Client-side: use injected env var or fallback
    return (process.env.NEXT_PUBLIC_API_URL as string | undefined) || 'http://localhost:10080/api'
  }
  // Server-side: use env var or fallback
  return process.env.NEXT_PUBLIC_API_URL || 'http://localhost:10080/api'
}

// Backend API'ye istek atmak için axios instance
export const api = axios.create({
  baseURL: getBaseURL(),
  withCredentials: true, // ✅ Cookie'leri otomatik gönder/al
})

// ❌ Token interceptor'ı kaldırdık (cookie otomatik gönderilir)

export default api