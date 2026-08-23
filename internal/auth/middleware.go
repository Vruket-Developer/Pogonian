package auth

import (
	"context"
	"net/http"
)

type contextKey string

const UserContextKey = contextKey("pogn_user")

// ContextWithUser stores the authenticated user in the request context.
func ContextWithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, UserContextKey, user)
}

// UserFromContext retrieves the authenticated user from the request context.
func UserFromContext(ctx context.Context) *User {
	user, _ := ctx.Value(UserContextKey).(*User)
	return user
}

// Middleware injects the authenticated user into request context if session exists.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := GlobalSessions.Get(r)
		if sess != nil && sess.User != nil {
			ctx := ContextWithUser(r.Context(), sess.User)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

// RequireAuth redirects unauthenticated requests to /login.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := GlobalSessions.Get(r)
		if sess == nil || sess.User == nil {
			http.Redirect(w, r, "/login?redirect="+r.URL.Path, http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireRole enforces specific role access (e.g. "admin").
func RequireRole(role string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := GlobalSessions.Get(r)
		if sess == nil || sess.User == nil || sess.User.Role != role {
			http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
