package products

import (
	"database/sql"
	"fmt"
	"time"
)

// Repository handles database operations for Products.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new Product repository and ensures the products table exists.
func NewRepository(db *sql.DB) (*Repository, error) {
	repo := &Repository{db: db}
	if err := repo.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate products table: %w", err)
	}
	return repo, nil
}

func (r *Repository) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		price REAL NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := r.db.Exec(query)
	return err
}

// FindAll returns all products from the database.
func (r *Repository) FindAll() ([]*Product, error) {
	rows, err := r.db.Query("SELECT id, name, description, price, created_at FROM products ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Product
	for rows.Next() {
		p := &Product{}
		var createdAtStr string
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &createdAtStr); err != nil {
			return nil, err
		}
		p.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		list = append(list, p)
	}
	return list, nil
}

// FindByID returns a single product by ID.
func (r *Repository) FindByID(id int64) (*Product, error) {
	row := r.db.QueryRow("SELECT id, name, description, price, created_at FROM products WHERE id = ?", id)
	p := &Product{}
	var createdAtStr string
	if err := row.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &createdAtStr); err != nil {
		return nil, err
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	return p, nil
}

// Create inserts a new product.
func (r *Repository) Create(name, desc string, price float64) (*Product, error) {
	res, err := r.db.Exec("INSERT INTO products (name, description, price, created_at) VALUES (?, ?, ?, ?)",
		name, desc, price, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Product{
		ID:          id,
		Name:        name,
		Description: desc,
		Price:       price,
		CreatedAt:   time.Now().UTC(),
	}, nil
}
