package request

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi"
)

const (
	RepoRefParamName = "rref"
)

func GetRepoRef(r *http.Request) (string, error) {
	rawRef := chi.URLParam(r, RepoRefParamName)
	if rawRef == "" {
		return "", errors.New("Repository ref parameter not found in request.")
	}

	// fqns are unescaped and lower
	ref, err := url.PathUnescape(rawRef)
	return strings.ToLower(ref), err
}
