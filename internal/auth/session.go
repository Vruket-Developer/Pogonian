package auth

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"
)

const SessionCookieName = "pogn_session"

// User represents an authenticated identity.
type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"` // "admin", "user"
	CreatedAt time.Time `json:"created_at"`
}

// Session represents an active authenticated session.
type Session struct {
	ID        string
	User      *User
	ExpiresAt time.Time
}

// SessionManager manages user sessions in memory.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	duration time.Duration
}

var GlobalSessions = NewSessionManager(24 * time.Hour)

// NewSessionManager creates a new session manager.
func NewSessionManager(duration time.Duration) *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
		duration: duration,
	}
}

// Create creates a new session and sets the HTTP session cookie.
func (sm *SessionManager) Create(w http.ResponseWriter, user *User) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	sessionID := hex.EncodeToString(b)

	sess := &Session{
		ID:        sessionID,
		User:      user,
		ExpiresAt: time.Now().Add(sm.duration),
	}
	sm.sessions[sessionID] = sess

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		Expires:  sess.ExpiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return sess, nil
}

// Get retrieves the active session from the request cookie.
func (sm *SessionManager) Get(r *http.Request) *Session {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil || cookie.Value == "" {
		return nil
	}

	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sess, exists := sm.sessions[cookie.Value]
	if !exists {
		return nil
	}

	if time.Now().After(sess.ExpiresAt) {
		return nil
	}

	return sess
}

// Destroy invalidates a session and clears the cookie.
func (sm *SessionManager) Destroy(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(SessionCookieName)
	if err == nil && cookie.Value != "" {
		sm.mu.Lock()
		delete(sm.sessions, cookie.Value)
		sm.mu.Unlock()
	}

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
	})
}
