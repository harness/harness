package web

import (
	"context"
	"crypto/md5"
	"fmt"
	"net/http"
	"time"

	"github.com/drone/drone-ui/dist"
	"github.com/drone/drone/model"
	"github.com/drone/drone/server/template"
	"github.com/drone/drone/shared/token"
	"github.com/drone/drone/version"

	"github.com/dimfeld/httptreemux"
)

// Endpoint provides the website endpoints.
type Endpoint interface {
	// Register registers the provider endpoints.
	Register(*httptreemux.ContextMux)
}

// New returns the default website endpoint.
func New() Endpoint {
	return new(website)
}

type website struct{}

func (w *website) Register(mux *httptreemux.ContextMux) {
	r := dist.New()
	h := http.FileServer(r)
	h = setupCache(h)
	mux.Handler("GET", "/favicon-32x32.png", h)
	mux.Handler("GET", "/favicon-16x16.png", h)
	mux.Handler("GET", "/src/*filepath", h)
	mux.Handler("GET", "/bower_components/*filepath", h)
	mux.NotFoundHandler = handleIndex
}

func handleIndex(rw http.ResponseWriter, r *http.Request) {
	rw.WriteHeader(200)

	var csrf string
	var user, _ = ToUser(r.Context())
	if user != nil {
		csrf, _ = token.New(
			token.CsrfToken,
			user.Login,
		).Sign(user.Hash)
	}
	params := map[string]interface{}{
		"user":    user,
		"csrf":    csrf,
		"version": version.Version.String(),
	}
	rw.Header().Set("Content-Type", "text/html; charset=UTF-8")
	template.T.ExecuteTemplate(rw, "index_polymer.html", params)
}

func setupCache(h http.Handler) http.Handler {
	data := []byte(time.Now().String())
	etag := fmt.Sprintf("%x", md5.Sum(data))

	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "public, max-age=31536000")
			w.Header().Del("Expires")
			w.Header().Set("ETag", etag)
			h.ServeHTTP(w, r)
		},
	)
}

func resetCache(h http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Del("Cache-Control")
			w.Header().Del("Last-Updated")
			w.Header().Del("Expires")
			w.Header().Del("ETag")
			h.ServeHTTP(w, r)
		},
	)
}

// WithUser returns a context with the current authenticated user.
func WithUser(c context.Context, user *model.User) context.Context {
	return context.WithValue(c, userKey, user)
}

// ToUser returns a user from the context.
func ToUser(c context.Context) (*model.User, bool) {
	user, ok := c.Value(userKey).(*model.User)
	return user, ok
}

type key int

const userKey key = 0
