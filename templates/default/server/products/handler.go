package products

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// Handler handles HTTP requests and form actions for Products.
type Handler struct {
	service *Service
}

// NewHandler creates a new Product HTTP Handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// HandleGetProducts returns the list of products for template rendering or JSON.
func (h *Handler) HandleGetProducts(r *http.Request, params map[string]string) (interface{}, error) {
	products, err := h.service.ListProducts()
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"Products": products,
		"Total":    len(products),
	}, nil
}

// HandleGetProductDetail returns a single product by ID param.
func (h *Handler) HandleGetProductDetail(r *http.Request, params map[string]string) (interface{}, error) {
	idStr := params["id"]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, err
	}

	product, err := h.service.GetProduct(id)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"Product": product,
	}, nil
}

// HandleCreateAction handles form POST submissions to create a new product.
func (h *Handler) HandleCreateAction(w http.ResponseWriter, r *http.Request, params map[string]string) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	name := r.FormValue("name")
	desc := r.FormValue("description")
	priceStr := r.FormValue("price")

	price, _ := strconv.ParseFloat(priceStr, 64)

	product, err := h.service.CreateProduct(name, desc, price)
	if err != nil {
		return err
	}

	// For AJAX / JSON requests
	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		return json.NewEncoder(w).Encode(product)
	}

	// For standard / soft-nav HTML forms, redirect back to /products
	http.Redirect(w, r, "/products", http.StatusSeeOther)
	return nil
}
