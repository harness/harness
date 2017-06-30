package server

import (
	"net/http"

	"github.com/drone/drone/model"
	"github.com/drone/drone/server/template"
	"github.com/drone/drone/shared/token"
)

type local struct {
	dir string
	fs  http.Handler
}

// NewLocalWebsite returns a new website loader.
func NewLocalWebsite(path string) Website {
	return &local{
		dir: path,
		fs: http.FileServer(
			http.Dir(path),
		),
	}
}

// Page serves a page in the user interface.
func (w *local) Page(rw http.ResponseWriter, r *http.Request, u *model.User) {
	rw.WriteHeader(200)

	path := r.URL.Path
	switch path {
	case "/login":
		if err := r.FormValue("error"); err != "" {
			// TODO login error
		} else {
			http.Redirect(rw, r, "/authorize", 303)
		}

	default:
		var csrf string
		if u != nil {
			csrf, _ = token.New(
				token.CsrfToken,
				u.Login,
			).Sign(u.Hash)
		}
		params := map[string]interface{}{
			"user": u,
			"csrf": csrf,
		}

		template.T.ExecuteTemplate(rw, "index_polymer.html", params)
	}
}

// File serves a static file for the user interface.
func (w *local) File(rw http.ResponseWriter, r *http.Request) {
	w.fs.ServeHTTP(rw, r)
}

func (w *local) Routes() []string {
	return []string{
		"/src/*filepath",
		"/bower_components/*filepath",
	}
}
