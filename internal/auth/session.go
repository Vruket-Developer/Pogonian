package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const SessionCookieName = "pogn_session"

var ErrSessionNotFound = errors.New("session not found or expired")

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
	ID        string    `json:"id"`
	User      *User     `json:"user"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SessionStore defines the pluggable persistence contract for sessions.
type SessionStore interface {
	Save(session *Session) error
	Get(sessionID string) (*Session, error)
	Delete(sessionID string) error
	Cleanup() error
}

// MemorySessionStore provides fast in-memory session caching.
type MemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{
		sessions: make(map[string]*Session),
	}
}

func (m *MemorySessionStore) Save(sess *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[sess.ID] = sess
	return nil
}

func (m *MemorySessionStore) Get(sessionID string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sess, exists := m.sessions[sessionID]
	if !exists || time.Now().After(sess.ExpiresAt) {
		return nil, ErrSessionNotFound
	}
	return sess, nil
}

func (m *MemorySessionStore) Delete(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sessionID)
	return nil
}

func (m *MemorySessionStore) Cleanup() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for id, sess := range m.sessions {
		if now.After(sess.ExpiresAt) {
			delete(m.sessions, id)
		}
	}
	return nil
}

// SQLiteSessionStore provides persistent cross-restart session storage in SQL databases.
type SQLiteSessionStore struct {
	db *sql.DB
}

func NewSQLiteSessionStore(db *sql.DB) (*SQLiteSessionStore, error) {
	stmt := `CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		user_email TEXT NOT NULL,
		user_name TEXT NOT NULL,
		user_role TEXT NOT NULL,
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	if _, err := db.Exec(stmt); err != nil {
		return nil, fmt.Errorf("failed to initialize sessions table: %w", err)
	}
	return &SQLiteSessionStore{db: db}, nil
}

func (s *SQLiteSessionStore) Save(sess *Session) error {
	stmt := `INSERT OR REPLACE INTO sessions (id, user_id, user_email, user_name, user_role, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(stmt,
		sess.ID,
		sess.User.ID,
		sess.User.Email,
		sess.User.Name,
		sess.User.Role,
		sess.ExpiresAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *SQLiteSessionStore) Get(sessionID string) (*Session, error) {
	stmt := `SELECT id, user_id, user_email, user_name, user_role, expires_at FROM sessions WHERE id = ?`
	row := s.db.QueryRow(stmt, sessionID)

	var sess Session
	var user User
	var expiresStr string

	if err := row.Scan(&sess.ID, &user.ID, &user.Email, &user.Name, &user.Role, &expiresStr); err != nil {
		return nil, ErrSessionNotFound
	}

	exp, err := time.Parse(time.RFC3339, expiresStr)
	if err != nil || time.Now().After(exp) {
		_ = s.Delete(sessionID)
		return nil, ErrSessionNotFound
	}

	sess.ExpiresAt = exp
	sess.User = &user
	return &sess, nil
}

func (s *SQLiteSessionStore) Delete(sessionID string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE id = ?", sessionID)
	return err
}

func (s *SQLiteSessionStore) Cleanup() error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now().UTC().Format(time.RFC3339))
	return err
}

// FileSessionStore provides persistent disk-backed JSON session storage.
type FileSessionStore struct {
	dir string
	mu  sync.RWMutex
}

func NewFileSessionStore(dir string) (*FileSessionStore, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %w", err)
	}
	return &FileSessionStore{dir: dir}, nil
}

func (f *FileSessionStore) sessionPath(id string) string {
	return filepath.Join(f.dir, "sess_"+id+".json")
}

func (f *FileSessionStore) Save(sess *Session) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := json.Marshal(sess)
	if err != nil {
		return err
	}
	return os.WriteFile(f.sessionPath(sess.ID), data, 0600)
}

func (f *FileSessionStore) Get(sessionID string) (*Session, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	path := f.sessionPath(sessionID)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}

	if time.Now().After(sess.ExpiresAt) {
		_ = os.Remove(path)
		return nil, ErrSessionNotFound
	}

	return &sess, nil
}

func (f *FileSessionStore) Delete(sessionID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	err := os.Remove(f.sessionPath(sessionID))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (f *FileSessionStore) Cleanup() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	entries, err := os.ReadDir(f.dir)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(f.dir, entry.Name())
		if data, err := os.ReadFile(path); err == nil {
			var sess Session
			if err := json.Unmarshal(data, &sess); err == nil && now.After(sess.ExpiresAt) {
				_ = os.Remove(path)
			}
		}
	}
	return nil
}

// SessionManager manages user sessions via a pluggable store.
type SessionManager struct {
	mu       sync.RWMutex
	store    SessionStore
	duration time.Duration
}

var GlobalSessions = NewSessionManager(NewMemorySessionStore(), 24*time.Hour)

// NewSessionManager creates a new session manager with a given store.
func NewSessionManager(store SessionStore, duration time.Duration) *SessionManager {
	return &SessionManager{
		store:    store,
		duration: duration,
	}
}

// SetStore dynamically upgrades or swaps the session store in a thread-safe manner.
func (sm *SessionManager) SetStore(store SessionStore) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.store = store
}

func (sm *SessionManager) getStore() SessionStore {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.store
}

// Create creates a new session and sets the HTTP session cookie.
func (sm *SessionManager) Create(w http.ResponseWriter, user *User) (*Session, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("failed to generate secure session ID: %w", err)
	}
	sessionID := hex.EncodeToString(b)

	sess := &Session{
		ID:        sessionID,
		User:      user,
		ExpiresAt: time.Now().Add(sm.duration),
	}

	if err := sm.getStore().Save(sess); err != nil {
		return nil, fmt.Errorf("failed to persist session: %w", err)
	}

	secure := os.Getenv("APP_ENV") == "production"

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		Expires:  sess.ExpiresAt,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})

	return sess, nil
}

// RegenerateSession creates a new session ID for an existing session (Session Fixation Prevention).
func (sm *SessionManager) RegenerateSession(w http.ResponseWriter, r *http.Request, user *User) (*Session, error) {
	sm.Destroy(w, r)
	return sm.Create(w, user)
}

// Get retrieves the active session from the request cookie.
func (sm *SessionManager) Get(r *http.Request) *Session {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil || cookie.Value == "" {
		return nil
	}

	sess, err := sm.getStore().Get(cookie.Value)
	if err != nil {
		return nil
	}
	return sess
}

// Destroy invalidates a session and clears the cookie.
func (sm *SessionManager) Destroy(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(SessionCookieName)
	if err == nil && cookie.Value != "" {
		_ = sm.getStore().Delete(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}
