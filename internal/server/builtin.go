package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"pogonian/internal/auth"
)

// setupBuiltinModules registers default data loaders and form actions for SQLite entities.
func (s *Server) setupBuiltinModules() {
	if s.DB == nil {
		return
	}

	// Auto-migrate standard users and demo entities table
	_ = auth.EnsureUsersTable(s.DB.DB)
	_ = s.DB.Migrate(`
		CREATE TABLE IF NOT EXISTS products (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT,
			price REAL NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)

	// Register Auth Actions
	s.RegisterAction("/register", auth.HandleRegisterAction(s.DB.DB))
	s.RegisterAction("/login", auth.HandleLoginAction(s.DB.DB))
	s.RegisterAction("/logout", auth.HandleLogoutAction())

	// 1. Data Loader for /products
	s.RegisterLoader("/products", func(r *http.Request, params map[string]string) (interface{}, error) {
		rows, err := s.DB.Query("SELECT id, name, description, price, created_at FROM products ORDER BY id DESC")
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		type ProductItem struct {
			ID          int64
			Name        string
			Description string
			Price       float64
			CreatedAt   string
		}

		var items []ProductItem
		for rows.Next() {
			var p ProductItem
			var rawTime string
			if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &rawTime); err != nil {
				continue
			}
			p.CreatedAt = rawTime
			items = append(items, p)
		}

		return map[string]interface{}{
			"Products": items,
			"Total":    len(items),
		}, nil
	})

	// 2. Data Loader for /products/:id
	s.RegisterLoader("/products/:id", func(r *http.Request, params map[string]string) (interface{}, error) {
		idStr := params["id"]
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid product ID: %s", idStr)
		}

		row := s.DB.QueryRow("SELECT id, name, description, price, created_at FROM products WHERE id = ?", id)
		type ProductDetail struct {
			ID          int64
			Name        string
			Description string
			Price       float64
			CreatedAt   string
		}

		var p ProductDetail
		var rawTime string
		if err := row.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &rawTime); err != nil {
			if err == sql.ErrNoRows {
				return map[string]interface{}{"Product": nil}, nil
			}
			return nil, err
		}
		p.CreatedAt = rawTime

		return map[string]interface{}{
			"Product": p,
		}, nil
	})

	// 3. Form Action POST for /products
	s.RegisterAction("/products", func(w http.ResponseWriter, r *http.Request, params map[string]string) error {
		_ = r.ParseMultipartForm(32 << 20)
		_ = r.ParseForm()

		name := r.FormValue("name")
		desc := r.FormValue("description")
		priceStr := r.FormValue("price")

		price, _ := strconv.ParseFloat(priceStr, 64)
		if name == "" {
			return fmt.Errorf("product name is required")
		}

		_, err := s.DB.Exec(
			"INSERT INTO products (name, description, price, created_at) VALUES (?, ?, ?, ?)",
			name, desc, price, time.Now().UTC().Format(time.RFC3339),
		)
		if err != nil {
			return fmt.Errorf("failed to insert product: %w", err)
		}

		// If AJAX request requesting JSON
		if r.Header.Get("Accept") == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			return json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		}

		// Standard HTML form submission -> redirect back to /products
		http.Redirect(w, r, "/products", http.StatusSeeOther)
		return nil
	})
}
