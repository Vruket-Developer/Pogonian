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

// RateLimiterOptions configures rate limiter behaviors.
type RateLimiterOptions struct {
	Limit       int
	Window      time.Duration
	TrustProxy  bool
	TrustedIPs  []string
}

// RateLimiter implements an in-memory rate limiter per client IP with proxy hardening.
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*clientVisitor
	opts     RateLimiterOptions
}

// NewRateLimiter creates a new RateLimiter.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return NewRateLimiterWithOptions(RateLimiterOptions{
		Limit:      limit,
		Window:     window,
		TrustProxy: false,
	})
}

// NewRateLimiterWithOptions creates a RateLimiter with explicit proxy trust settings.
func NewRateLimiterWithOptions(opts RateLimiterOptions) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*clientVisitor),
		opts:     opts,
	}

	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) cleanup() {
	for {
		time.Sleep(rl.opts.Window * 2)
		rl.mu.Lock()
		now := time.Now()
		for ip, v := range rl.visitors {
			if now.Sub(v.lastSeen) > rl.opts.Window {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimit returns a middleware enforcing request rate limits.
func RateLimit(limit int, window time.Duration) Middleware {
	rl := NewRateLimiter(limit, window)
	return rl.Handler()
}

// Handler returns the HTTP middleware handler.
func (rl *RateLimiter) Handler() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := rl.extractIP(r)

			rl.mu.Lock()
			v, exists := rl.visitors[ip]
			now := time.Now()

			if !exists || now.Sub(v.lastSeen) > rl.opts.Window {
				rl.visitors[ip] = &clientVisitor{lastSeen: now, count: 1}
				rl.mu.Unlock()
				next.ServeHTTP(w, r)
				return
			}

			v.count++
			v.lastSeen = now

			if v.count > rl.opts.Limit {
				rl.mu.Unlock()
				http.Error(w, "429 Too Many Requests: Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			rl.mu.Unlock()
			next.ServeHTTP(w, r)
		})
	}
}

func (rl *RateLimiter) extractIP(r *http.Request) string {
	// Only inspect X-Forwarded-For if explicitly configured to trust upstream proxy
	if rl.opts.TrustProxy {
		xff := r.Header.Get("X-Forwarded-For")
		if xff != "" {
			parts := strings.Split(xff, ",")
			clientIP := strings.TrimSpace(parts[0])
			if clientIP != "" {
				return clientIP
			}
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
