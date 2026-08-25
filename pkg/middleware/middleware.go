package middleware

import "net/http"

// Middleware defines a standard HTTP middleware wrapper function.
type Middleware func(http.Handler) http.Handler

// Chain combines multiple middlewares into a single handler.
// Middlewares are executed in the order they are passed.
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
