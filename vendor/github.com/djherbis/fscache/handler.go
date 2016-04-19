package fscache

import (
	"io"
	"net/http"
)

// Handler is a caching middle-ware for http Handlers.
// It responds to http requests via the passed http.Handler, and caches the response
// using the passed cache. The cache key for the request is the req.URL.String().
// Note: It does not cache http headers. It is more efficient to set them yourself.
func Handler(c Cache, h http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		url := req.URL.String()
		r, w, err := c.Get(url)
		if err != nil {
			h.ServeHTTP(rw, req)
			return
		}
		defer r.Close()
		if w != nil {
			go func() {
				defer w.Close()
				h.ServeHTTP(&respWrapper{
					ResponseWriter: rw,
					Writer:         w,
				}, req)
			}()
		}
		io.Copy(rw, r)
	})
}

type respWrapper struct {
	http.ResponseWriter
	io.Writer
}

func (r *respWrapper) Write(p []byte) (int, error) {
	return r.Writer.Write(p)
}
