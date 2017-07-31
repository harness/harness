package web

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"path/filepath"

	"github.com/drone/drone/server/template"
	"github.com/drone/drone/shared/token"
	"github.com/drone/drone/version"

	"github.com/dimfeld/httptreemux"
)

// FromPath returns the default website endpoint served from a local path.
func FromPath(path string) Endpoint {
	return &local{path}
}

type local struct {
	dir string
}

func (l *local) Register(mux *httptreemux.ContextMux) {
	h := http.FileServer(
		http.Dir(l.dir),
	)
	h = resetCache(h)
	mux.Handler("GET", "/favicon-32x32.png", h)
	mux.Handler("GET", "/favicon-16x16.png", h)
	mux.Handler("GET", "/src/*filepath", h)
	mux.Handler("GET", "/static/*filepath", h)
	mux.Handler("GET", "/bundle/*filepath", h)
	mux.Handler("GET", "/bower_components/*filepath", h)
	mux.NotFoundHandler = l.handleIndexLocal
}

func (l *local) handleIndexLocal(rw http.ResponseWriter, r *http.Request) {
	var csrf string

	user, _ := ToUser(r.Context())
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

	index, err := ioutil.ReadFile(filepath.Join(l.dir, "index.html"))
	if err != nil {
		rw.WriteHeader(404)
		return
	}
	var buf bytes.Buffer
	template.T.ExecuteTemplate(&buf, "script.html", params)
	index = bytes.Replace(index, []byte("<!-- inject:js -->"), buf.Bytes(), 1)
	rw.WriteHeader(200)
	rw.Write(index)
}
