package render

import (
	"html/template"
	"io"
	"net/http"
)

// Render applies the template that has the given name to the specified data
// object and writes the output to wr.
type Render func(wr io.Writer, name string, data interface{}) error

type Renderer interface {
	// HTML renders the named HTML template
	HTML(wr io.Writer, name string, data interface{}) error

	// NotFound renders the 404 HTML template. It also writes the
	// appropriate response status if io.Writer is of type http.ResponseWriter.
	NotFound(wr io.Writer, data interface{}) error

	// NotAuthorized renders the 401 HTML template. It also writes the
	// appropriate response status if io.Writer is of type http.ResponseWriter.
	NotAuthorized(wr io.Writer, data interface{}) error
}

type renderer struct {
	*template.Template
}

func NewRenderer(t *template.Template) Renderer {
	return &renderer{t}
}

// HTML renders the named HTML template
func (r *renderer) HTML(w io.Writer, name string, data interface{}) error {
	return r.ExecuteTemplate(w, name, data)
}

// NotFound renders the 404 HTML template. It also writes the
// appropriate response status if io.Writer is of type http.ResponseWriter.
func (r *renderer) NotFound(w io.Writer, data interface{}) error {
	if rw, ok := w.(http.ResponseWriter); !ok {
		rw.WriteHeader(http.StatusNotFound)
	}
	return r.HTML(w, "404.html", data)
}

// NotAuthorized renders the 401 HTML template. It also writes the
// appropriate response status if io.Writer is of type http.ResponseWriter.
func (r *renderer) NotAuthorized(w io.Writer, data interface{}) error {
	if rw, ok := w.(http.ResponseWriter); !ok {
		rw.WriteHeader(http.StatusUnauthorized)
	}
	return r.HTML(w, "401.html", data)
}
