package server

import (
	"pogonian/internal/auth"
)

// setupBuiltinModules registers foundational authentication actions and infrastructure tables.
func (s *Server) setupBuiltinModules() {
	if s.DB == nil {
		return
	}

	// Auto-migrate standard user authentication table
	_ = auth.EnsureUsersTable(s.DB.DB)

	// Upgrade global session manager to SQLite persistent store if available
	if s.DB.DB != nil {
		if sqStore, err := auth.NewSQLiteSessionStore(s.DB.DB); err == nil {
			auth.GlobalSessions.SetStore(sqStore)
		}
	}

	// Register Core Auth Action Handlers
	s.RegisterAction("/register", auth.HandleRegisterAction(s.DB.DB))
	s.RegisterAction("/login", auth.HandleLoginAction(s.DB.DB))
	s.RegisterAction("/logout", auth.HandleLogoutAction())
}
