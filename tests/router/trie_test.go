package router_test

import (
	"testing"

	"pogonian/pkg/router"
)

func TestRouter_TrieMatching(t *testing.T) {
	trie := router.NewTrieRouter()

	homeRoute := &router.Route{Path: "/"}
	productsRoute := &router.Route{Path: "/products"}
	productDetailRoute := &router.Route{Path: "/products/:id"}
	userProfileRoute := &router.Route{Path: "/users/[username]"}

	trie.Insert("/", homeRoute)
	trie.Insert("/products", productsRoute)
	trie.Insert("/products/[id]", productDetailRoute)
	trie.Insert("/users/[username]", userProfileRoute)

	// 1. Exact root match
	r, params := trie.Match("/")
	if r != homeRoute {
		t.Errorf("Expected root route, got %v", r)
	}
	if len(params) != 0 {
		t.Errorf("Expected 0 params for root, got %d", len(params))
	}

	// 2. Exact static match
	r, params = trie.Match("/products")
	if r != productsRoute {
		t.Errorf("Expected products route, got %v", r)
	}

	// 3. Dynamic parameter match
	r, params = trie.Match("/products/42")
	if r != productDetailRoute {
		t.Errorf("Expected product detail route, got %v", r)
	}
	if params["id"] != "42" {
		t.Errorf("Expected param id='42', got '%s'", params["id"])
	}

	// 4. Custom parameter name
	r, params = trie.Match("/users/alex")
	if r != userProfileRoute {
		t.Errorf("Expected user profile route, got %v", r)
	}
	if params["username"] != "alex" {
		t.Errorf("Expected param username='alex', got '%s'", params["username"])
	}

	// 5. Unknown route returns nil
	r, _ = trie.Match("/non/existent/path")
	if r != nil {
		t.Errorf("Expected nil for non-existent path, got %v", r)
	}
}
