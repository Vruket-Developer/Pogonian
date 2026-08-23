package auth

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// EnsureUsersTable creates the users table in SQLite if it doesn't exist.
func EnsureUsersTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		name TEXT,
		role TEXT DEFAULT 'user',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	_, err := db.Exec(query)
	return err
}

// Register creates a new user account with hashed password.
func Register(db *sql.DB, email, password, name, role string) (*User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || password == "" {
		return nil, errors.New("email and password are required")
	}
	if role == "" {
		role = "user"
	}

	hash, err := HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now().UTC()
	res, err := db.Exec(
		"INSERT INTO users (email, password_hash, name, role, created_at) VALUES (?, ?, ?, ?, ?)",
		email, hash, name, role, now.Format(time.RFC3339),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return nil, errors.New("an account with this email already exists")
		}
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &User{
		ID:        id,
		Email:     email,
		Name:      name,
		Role:      role,
		CreatedAt: now,
	}, nil
}

// Login verifies credentials and returns the matching user.
func Login(db *sql.DB, email, password string) (*User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || password == "" {
		return nil, errors.New("email and password are required")
	}

	row := db.QueryRow("SELECT id, email, password_hash, name, role, created_at FROM users WHERE email = ?", email)
	var (
		id        int64
		userEmail string
		hash      string
		name      string
		role      string
		timeStr   string
	)

	if err := row.Scan(&id, &userEmail, &hash, &name, &role, &timeStr); err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("invalid email or password")
		}
		return nil, err
	}

	if !CheckPasswordHash(password, hash) {
		return nil, errors.New("invalid email or password")
	}

	createdAt, _ := time.Parse(time.RFC3339, timeStr)
	return &User{
		ID:        id,
		Email:     userEmail,
		Name:      name,
		Role:      role,
		CreatedAt: createdAt,
	}, nil
}

// HandleRegisterAction handles POST /register
func HandleRegisterAction(db *sql.DB) func(w http.ResponseWriter, r *http.Request, params map[string]string) error {
	return func(w http.ResponseWriter, r *http.Request, params map[string]string) error {
		_ = r.ParseMultipartForm(32 << 20)
		_ = r.ParseForm()

		email := r.FormValue("email")
		password := r.FormValue("password")
		name := r.FormValue("name")

		user, err := Register(db, email, password, name, "user")
		if err != nil {
			return err
		}

		if _, err := GlobalSessions.Create(w, user); err != nil {
			return err
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return nil
	}
}

// HandleLoginAction handles POST /login
func HandleLoginAction(db *sql.DB) func(w http.ResponseWriter, r *http.Request, params map[string]string) error {
	return func(w http.ResponseWriter, r *http.Request, params map[string]string) error {
		_ = r.ParseMultipartForm(32 << 20)
		_ = r.ParseForm()

		email := r.FormValue("email")
		password := r.FormValue("password")

		user, err := Login(db, email, password)
		if err != nil {
			return err
		}

		if _, err := GlobalSessions.Create(w, user); err != nil {
			return err
		}

		redirectUrl := r.URL.Query().Get("redirect")
		if redirectUrl == "" {
			redirectUrl = "/"
		}
		http.Redirect(w, r, redirectUrl, http.StatusSeeOther)
		return nil
	}
}

// HandleLogoutAction handles POST /logout
func HandleLogoutAction() func(w http.ResponseWriter, r *http.Request, params map[string]string) error {
	return func(w http.ResponseWriter, r *http.Request, params map[string]string) error {
		GlobalSessions.Destroy(w, r)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return nil
	}
}
