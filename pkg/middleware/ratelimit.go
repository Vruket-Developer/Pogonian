package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type clientVisitor struct {
	lastSeen time.Time
	count    int
}

// RateLimiter implements a lightweight in-memory rate limiter per IP.
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*clientVisitor
	limit    int
	window   time.Duration
}

// NewRateLimiter creates a new RateLimiter.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*clientVisitor),
		limit:    limit,
		window:   window,
	}

	go rl.cleanup()

	return rl
}

func (rl *RateLimiter) cleanup() {
	for {
		time.Sleep(rl.window * 2)
		rl.mu.Lock()
		now := time.Now()
		for ip, v := range rl.visitors {
			if now.Sub(v.lastSeen) > rl.window {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimit returns a middleware enforcing request rate limits.
func RateLimit(limit int, window time.Duration) Middleware {
	rl := NewRateLimiter(limit, window)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractIP(r)

			rl.mu.Lock()
			v, exists := rl.visitors[ip]
			now := time.Now()

			if !exists || now.Sub(v.lastSeen) > rl.window {
				rl.visitors[ip] = &clientVisitor{lastSeen: now, count: 1}
				rl.mu.Unlock()
				next.ServeHTTP(w, r)
				return
			}

			v.count++
			v.lastSeen = now

			if v.count > rl.limit {
				rl.mu.Unlock()
				http.Error(w, "429 Too Many Requests: Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			rl.mu.Unlock()
			next.ServeHTTP(w, r)
		})
	}
}

func extractIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
