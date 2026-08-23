package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
)

// Recovery catches panics, logs stack traces, and returns a clean 500 error page.
func Recovery(devMode bool) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					stack := debug.Stack()
					fmt.Printf("\x1b[1;31m[PANIC RECOVERED]\x1b[0m %v\n%s\n", rec, string(stack))

					w.WriteHeader(http.StatusInternalServerError)
					w.Header().Set("Content-Type", "text/html; charset=utf-8")

					if devMode {
						fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>500 Internal Server Error</title><link rel="stylesheet" href="/globals.css"></head>
<body class="pogn-body" style="padding: 3rem;">
    <h1 style="color: #ef4444;">Server Error (Panic)</h1>
    <p><strong>Reason:</strong> %v</p>
    <pre style="background: rgba(0,0,0,0.5); padding: 1.5rem; border-radius: 8px; overflow-x: auto; color: #f87171;">%s</pre>
</body>
</html>`, rec, string(stack))
					} else {
						fmt.Fprintf(w, "<h1>500 Internal Server Error</h1><p>Something went wrong on the server.</p>")
					}
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
