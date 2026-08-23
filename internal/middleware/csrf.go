package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
)

const (
	CSRFCookieName = "_pogn_csrf"
	CSRFHeaderName = "X-CSRF-Token"
	CSRFFormField  = "_csrf"
)

// GenerateToken generates a cryptographically secure 32-byte hexadecimal CSRF token, failing closed if randomness fails.
func GenerateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("cryptographic randomness failure: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// CSRF provides Cross-Site Request Forgery protection that is enabled by default and fails closed.
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
				newToken, err := GenerateToken()
				if err != nil {
					http.Error(w, "500 Internal Server Error: Failed to generate CSRF token", http.StatusInternalServerError)
					return
				}
				token = newToken
				http.SetCookie(w, &http.Cookie{
					Name:     CSRFCookieName,
					Value:    token,
					Path:     "/",
					HttpOnly: false, // accessible to client runtime
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

			// For state-changing methods (POST, PUT, DELETE, PATCH), validate token strictly
			receivedToken := r.Header.Get(CSRFHeaderName)

			// If not in header, check form body or query
			if receivedToken == "" {
				_ = r.ParseMultipartForm(32 << 20)
				_ = r.ParseForm()
				receivedToken = r.FormValue(CSRFFormField)
			}

			// Constant-time token verification (Fail Closed)
			if receivedToken == "" || subtle.ConstantTimeCompare([]byte(receivedToken), []byte(token)) != 1 {
				http.Error(w, "403 Forbidden: Invalid or missing CSRF token", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
