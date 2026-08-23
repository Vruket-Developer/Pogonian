package auth_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pogonian/internal/auth"
	_ "modernc.org/sqlite"
)

func TestAuth_PasswordHashing(t *testing.T) {
	password := "SecretP@ssw0rd!123"
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if !auth.CheckPasswordHash(password, hash) {
		t.Errorf("Expected password to match hash")
	}

	if auth.CheckPasswordHash("WrongPassword", hash) {
		t.Errorf("Expected wrong password to fail check")
	}
}

func TestAuth_UserRegistrationAndLogin(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open sqlite: %v", err)
	}
	defer db.Close()

	if err := auth.EnsureUsersTable(db); err != nil {
		t.Fatalf("Failed to create users table: %v", err)
	}

	// 1. Register User
	user, err := auth.Register(db, "alex@example.com", "pass12345", "Alex Developer", "admin")
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}
	if user.Email != "alex@example.com" || user.Role != "admin" {
		t.Errorf("User fields mismatch: %+v", user)
	}

	// 2. Duplicate Registration Rejection
	_, err = auth.Register(db, "alex@example.com", "pass12345", "Alex", "user")
	if err == nil {
		t.Errorf("Expected duplicate email to fail")
	}

	// 3. Login Valid Credentials
	loggedIn, err := auth.Login(db, "alex@example.com", "pass12345")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}
	if loggedIn.ID != user.ID {
		t.Errorf("Expected user ID %d, got %d", user.ID, loggedIn.ID)
	}

	// 4. Login Invalid Credentials
	_, err = auth.Login(db, "alex@example.com", "wrongpassword")
	if err == nil {
		t.Errorf("Expected login with wrong password to fail")
	}
}

func TestAuth_SessionAndRouteGuard(t *testing.T) {
	sm := auth.NewSessionManager(1 * time.Hour)

	user := &auth.User{
		ID:    1,
		Email: "admin@example.com",
		Name:  "Admin",
		Role:  "admin",
	}

	// Protected handler
	protected := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("SECRET_DASHBOARD"))
	})

	guarded := auth.RequireAuth(protected)

	// 1. Unauthorized request without session cookie -> redirect to /login
	req := httptest.NewRequest("GET", "/dashboard", nil)
	rec := httptest.NewRecorder()
	guarded.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("Expected status 303 redirect, got %d", rec.Code)
	}
	if rec.Header().Get("Location") != "/login?redirect=/dashboard" {
		t.Errorf("Expected redirect to /login?redirect=/dashboard, got %s", rec.Header().Get("Location"))
	}

	// 2. Create session and make request with cookie
	recLogin := httptest.NewRecorder()
	sess, err := sm.Create(recLogin, user)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	reqWithCookie := httptest.NewRequest("GET", "/dashboard", nil)
	reqWithCookie.AddCookie(&http.Cookie{
		Name:  auth.SessionCookieName,
		Value: sess.ID,
	})

	recAuthed := httptest.NewRecorder()
	auth.GlobalSessions = sm
	guarded.ServeHTTP(recAuthed, reqWithCookie)

	if recAuthed.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for authenticated user, got %d", recAuthed.Code)
	}
}
