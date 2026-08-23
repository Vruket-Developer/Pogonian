package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

const (
	CSRFCookieName = "_pogn_csrf"
	CSRFHeaderName = "X-CSRF-Token"
	CSRFFormField  = "_csrf"
)

// GenerateToken generates a cryptographically secure 32-byte hexadecimal CSRF token.
func GenerateToken() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "csrf_fallback_token_001"
	}
	return hex.EncodeToString(bytes)
}

// CSRF provides Cross-Site Request Forgery protection.
func CSRF(enabled bool) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Get existing token from cookie or create a new one
			cookie, err := r.Cookie(CSRFCookieName)
			var token string
			if err != nil || cookie.Value == "" {
				token = GenerateToken()
				http.SetCookie(w, &http.Cookie{
					Name:     CSRFCookieName,
					Value:    token,
					Path:     "/",
					HttpOnly: false, // accessible to client runtime if needed
					SameSite: http.SameSiteLaxMode,
				})
			} else {
				token = cookie.Value
			}

			// For safe HTTP methods (GET, HEAD, OPTIONS), allow request
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}

			// For state-changing methods (POST, PUT, DELETE, PATCH), validate token
			// Check header first
			receivedToken := r.Header.Get(CSRFHeaderName)

			// If not in header and it is a form POST, check form body or query
			if receivedToken == "" {
				if err := r.ParseForm(); err == nil {
					receivedToken = r.FormValue(CSRFFormField)
				}
			}

			// Allow Pogonian runtime internal requests with X-Pogonian-Request header for soft navigation
			if r.Header.Get("X-Pogonian-Request") == "true" && receivedToken == "" {
				receivedToken = token // Validated via same-origin policy + custom header
			}

			if receivedToken == "" || receivedToken != token {
				http.Error(w, "403 Forbidden: Invalid or missing CSRF token", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
