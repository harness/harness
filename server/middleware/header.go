package middleware

import (
	"net/http"
	"time"

	"github.com/zenazn/goji/web"
)

// SetHeaders is a middleware function that applies
// default headers and caching rules to each request.
func SetHeaders(c *web.C, h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "*")
		w.Header().Add("X-Frame-Options", "DENY")
		w.Header().Add("X-Content-Type-Options", "nosniff")
		w.Header().Add("X-XSS-Protection", "1; mode=block")
		w.Header().Add("Cache-Control", "no-cache")
		w.Header().Add("Cache-Control", "no-store")
		w.Header().Add("Cache-Control", "max-age=0")
		w.Header().Add("Cache-Control", "must-revalidate")
		w.Header().Add("Cache-Control", "value")
		w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		w.Header().Set("Expires", "Thu, 01 Jan 1970 00:00:00 GMT")
		if r.TLS != nil {
			w.Header().Add("Strict-Transport-Security", "max-age=31536000")
		}
		h.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
