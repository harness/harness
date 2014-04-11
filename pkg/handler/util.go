package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dchest/authcookie"
	"github.com/drone/drone/pkg/template"
)

// -----------------------------------------------------------------------------
// Rendering Functions

// Renders the named template for the specified data type
// and write the output to the http.ResponseWriter.
func RenderTemplate(w http.ResponseWriter, name string, data interface{}) error {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	return template.ExecuteTemplate(w, name, data)
}

// Renders the 404 template for the specified data type
// and write the output to the http.ResponseWriter.
func RenderNotFound(w http.ResponseWriter) error {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	return template.ExecuteTemplate(w, "404.amber", nil)
}

// Renders the 403 template for the specified data type
// and write the output to the http.ResponseWriter.
func RenderForbidden(w http.ResponseWriter) error {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	return template.ExecuteTemplate(w, "403.amber", nil)
}

// RenderJson renders a JSON representation of resource v and
// writes to the http.ResposneWriter.
func RenderJson(w http.ResponseWriter, data interface{}) error {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(data)
}

// RenderText write the plain text string to the http.ResposneWriter.
func RenderText(w http.ResponseWriter, text string, code int) error {
	w.Header().Add("Content-Type", "text/plain")
	w.WriteHeader(code)
	w.Write([]byte(text))
	return nil
}

// RenderError renders a text representation of the Error message.
func RenderError(w http.ResponseWriter, err error, code int) error {
	return RenderText(w, err.Error(), code)
}

// -----------------------------------------------------------------------------
// Cookie Helper functions

// SetCookie signs and writes the cookie value.
func SetCookie(w http.ResponseWriter, r *http.Request, name, value string) {
	sec := IsHttps(r)
	str := authcookie.New(value, time.Now().Add(time.Hour*24), secret)
	cookie := http.Cookie{
		Name:     name,
		Value:    str,
		Path:     "/",
		Domain:   r.URL.Host,
		HttpOnly: true,
		Secure:   sec,
	}

	http.SetCookie(w, &cookie)
}

func IsHttps(r *http.Request) bool {
	if r.URL.Scheme == "https" {
		return true
	}
	if strings.HasPrefix(r.Proto, "HTTPS") {
		return true
	}
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		return true
	}
	return false
}

// GetCookie retrieves and verifies the signed cookie value.
func GetCookie(r *http.Request, name string) string {
	cookie, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return authcookie.Login(cookie.Value, secret)
}

// DelCookie deletes a secure cookie.
func DelCookie(w http.ResponseWriter, r *http.Request, name string) {
	cookie := http.Cookie{
		Name:   name,
		Value:  "deleted",
		Path:   "/",
		Domain: r.URL.Host,
		MaxAge: -1,
	}

	http.SetCookie(w, &cookie)
}

// FileFetchingProxy is a reverse proxy for a single file that
// can rewrite outgoing headers, and obeys If-Modified-Since
type FileFetchingProxy struct {
	Url            string
	HeaderRewrites map[string]string
	ModTime        time.Time
}

func (p *FileFetchingProxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Give the client NotModified if ModTime is provided and the client asks
	ifModTime := req.Header.Get("If-Modified-Since")
	if !p.ModTime.IsZero() && ifModTime != "" {
		t, err := time.Parse(http.TimeFormat, ifModTime)
		if err == nil && p.ModTime.After(t) {
			rw.WriteHeader(http.StatusNotModified)
			return
		}
	}
	res, err := http.Get(p.Url)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()
	for k, v := range res.Header {
		for _, vv := range v {
			rw.Header().Add(k, vv)
		}
	}
	for k, v := range p.HeaderRewrites {
		rw.Header().Set(k, v)
	}
	rw.WriteHeader(res.StatusCode)
	io.Copy(rw, res.Body)
}
