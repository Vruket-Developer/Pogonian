package runtime

import (
	_ "embed"
	"net/http"
)

//go:embed runtime.js
var RuntimeJS []byte

// Handler serves the Pogonian client runtime script.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Write(RuntimeJS)
	})
}
