package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeadersMiddleware adds security headers to all responses
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Prevent MIME-sniffing attacks
		// Browser'ın dosya tipini tahmin etmesini engeller
		// XSS saldırılarını önler (.jpg içinde script çalıştıramaz)
		c.Header("X-Content-Type-Options", "nosniff")

		// 2. Prevent clickjacking attacks
		// Site'nin iframe içinde açılmasını engeller
		// Kullanıcı senin siteni görmeden başka bir site üzerinden işlem yapamaz
		c.Header("X-Frame-Options", "DENY")

		// 3. XSS protection (legacy browsers için)
		// Modern browser'larda CSP daha etkili
		c.Header("X-XSS-Protection", "1; mode=block")

		// 4. Content Security Policy (XSS koruması)
		// Sadece kendi domaininden script çalışabilir
		// Inline script'ler ve external script'ler engellenir
		c.Header("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self'; "+
				"style-src 'self' 'unsafe-inline'; "+ // Tailwind için unsafe-inline gerekebilir
				"img-src 'self' data: https:; "+
				"font-src 'self'; "+
				"connect-src 'self' ws: wss:; "+ // WebSocket için ws/wss izin ver
				"frame-ancestors 'none';", // iframe'de açılmayı engelle
		)

		// 5. Referrer Policy
		// Hangi bilgilerin external sitelere gönderileceğini kontrol et
		// Kullanıcı privacy'sini korur
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// 6. Permissions Policy (önceden Feature-Policy)
		// Tarayıcı özelliklerini kısıtla (kamera, mikrofon, vb.)
		c.Header("Permissions-Policy",
			"camera=(), microphone=(), geolocation=(), payment=()",
		)

		c.Next()
	}
}

// HSTSMiddleware enforces HTTPS (only for production)
func HSTSMiddleware(isProduction bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isProduction {
			// HTTP Strict Transport Security
			// Browser'a "bu site SADECE HTTPS kullanır" der
			// max-age: 1 yıl (31536000 saniye)
			// includeSubDomains: Alt domainler de HTTPS kullanmalı
			// preload: Browser'ın HSTS preload listesine eklenebilir
			c.Header("Strict-Transport-Security",
				"max-age=31536000; includeSubDomains; preload",
			)
		}
		c.Next()
	}
}
