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
func New() Endpoint {
	return &website{
		fs: dist.New(),
		templ: mustCreateTemplate(
			string(dist.MustLookup("/index.html")),
		),
	}
}

// FromPath returns the website endpoint that
// serves the webpage form disk at path p.
func FromPath(p string) Endpoint {
	f := filepath.Join(p, "index.html")
	b, err := ioutil.ReadFile(f)
	if err != nil {
		panic(err)
	}
	return &website{
		fs:    http.Dir(p),
		templ: mustCreateTemplate(string(b)),
	}
}

type website struct {
	fs    http.FileSystem
	templ *template.Template
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
	params := map[string]interface{}{
		"user":    user,
		"csrf":    csrf,
		"version": version.Version.String(),
	}
	rw.Header().Set("Content-Type", "text/html; charset=UTF-8")

	w.templ.Execute(rw, params)
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

// var partials = templ
// var templ = `
// {{define "user"}}
// <script>
// 	{{ if .user }}
// 	window.USER = {{ json .user }};
// 	{{ end }}
// </script>
// {{end}}
//
// {{define "csrf"}}
// <script>
// 	{{ if .csrf }}window.DRONE_CSRF = "{{ .csrf }}"{{ end }}
// </script>
// {{end}}
//
// {{define "version"}}
// 	<meta name="version" content="{{ .version }}">
// {{end}}
// `

// var funcMap = template.FuncMap{"json": marshal}
//
// // marshal is a template helper function to render data as json.
// func marshal(v interface{}) template.JS {
// 	a, _ := json.Marshal(v)
// 	return template.JS(a)
// }
