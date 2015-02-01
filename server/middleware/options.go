package middleware

import (
	"net/http"

	"github.com/zenazn/goji/web"
)

// Options automatically return an appropriate "Allow" header when the
// request method is OPTIONS and the request would have otherwise been 404'd.
func Options(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization")
			w.Header().Set("Allow", "HEAD,GET,POST,PUT,DELETE,OPTIONS")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			return
		}
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
