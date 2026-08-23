package products

import (
	"errors"
	"strings"
)

// Service encapsulates business logic for Products.
type Service struct {
	repo *Repository
}

// NewService creates a new Product Service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// ListProducts retrieves all available products.
func (s *Service) ListProducts() ([]*Product, error) {
	return s.repo.FindAll()
}

// GetProduct retrieves a product by its unique ID.
func (s *Service) GetProduct(id int64) (*Product, error) {
	if id <= 0 {
		return nil, errors.New("invalid product ID")
	}
	return s.repo.FindByID(id)
}

// CreateProduct validates input and creates a product.
func (s *Service) CreateProduct(name, desc string, price float64) (*Product, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("product name cannot be empty")
	}
	if price < 0 {
		return nil, errors.New("product price must be positive")
	}

	return s.repo.Create(name, desc, price)
}
