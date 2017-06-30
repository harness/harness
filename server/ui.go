package server

import (
	"net/http"
	"os"

	"github.com/drone/drone-ui/dist"
	"github.com/drone/drone/model"
	"github.com/drone/drone/server/template"
	"github.com/drone/drone/shared/token"
)

// Website defines an interface to serve the user interface.
type Website interface {
	Page(rw http.ResponseWriter, r *http.Request, u *model.User)
	File(rw http.ResponseWriter, r *http.Request)
	Routes() []string
}

type website struct {
	fs http.Handler
}

// NewWebsite returns a new website loader.
func NewWebsite() Website {
	path := os.Getenv("DRONE_WWW")
	if path != "" {
		return NewLocalWebsite(path)
	}
	return &website{
		fs: http.FileServer(dist.AssetFS()),
	}
}

// Page serves a page in the user interface.
func (w *website) Page(rw http.ResponseWriter, r *http.Request, u *model.User) {
	rw.WriteHeader(200)

	path := r.URL.Path
	switch path {
	case "/login/form":
		params := map[string]interface{}{}
		template.T.ExecuteTemplate(rw, "login.html", params)

	case "/login":
		if err := r.FormValue("error"); err != "" {
			params := map[string]interface{}{"error": err}
			template.T.ExecuteTemplate(rw, "error.html", params)
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
		template.T.ExecuteTemplate(rw, "index.html", params)
	}
}

// File serves a static file for the user interface.
func (w *website) File(rw http.ResponseWriter, r *http.Request) {
	w.fs.ServeHTTP(rw, r)
}

func (w *website) Routes() []string {
	return []string{
		"/static/*filepath",
	}
}
