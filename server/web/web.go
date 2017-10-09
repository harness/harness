package web

import (
	"context"
	"crypto/md5"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"time"

	"github.com/drone/drone-ui/dist"
	"github.com/drone/drone/model"
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
func New(opt ...Option) Endpoint {
	opts := new(Options)
	for _, f := range opt {
		f(opts)
	}

	if opts.path != "" {
		return fromPath(opts)
	}

	return &website{
		fs:   dist.New(),
		opts: opts,
		tmpl: mustCreateTemplate(
			string(dist.MustLookup("/index.html")),
		),
	}
}

func fromPath(opts *Options) *website {
	f := filepath.Join(opts.path, "index.html")
	b, err := ioutil.ReadFile(f)
	if err != nil {
		panic(err)
	}
	return &website{
		fs:   http.Dir(opts.path),
		tmpl: mustCreateTemplate(string(b)),
		opts: opts,
	}
}

type website struct {
	opts *Options
	fs   http.FileSystem
	tmpl *template.Template
}

func (w *website) Register(mux *httptreemux.ContextMux) {
	h := http.FileServer(w.fs)
	h = setupCache(h)
	mux.Handler("GET", "/favicon.png", h)
	mux.Handler("GET", "/static/*filepath", h)
	mux.NotFoundHandler = w.handleIndex
}

func (w *website) handleIndex(rw http.ResponseWriter, r *http.Request) {
	rw.WriteHeader(200)

	var csrf string
	var user, _ = ToUser(r.Context())
	if user != nil {
		csrf, _ = token.New(
			token.CsrfToken,
			user.Login,
		).Sign(user.Hash)
	}
	var syncing bool
	if user != nil {
		syncing = time.Unix(user.Synced, 0).Add(w.opts.sync).Before(time.Now())
	}
	params := map[string]interface{}{
		"user":    user,
		"csrf":    csrf,
		"syncing": syncing,
		"version": version.Version.String(),
	}
	rw.Header().Set("Content-Type", "text/html; charset=UTF-8")

	w.tmpl.Execute(rw, params)
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
