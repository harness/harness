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

var (
	ErrRepoReferenceNotFound = errors.New("No repository reference found in request.")
)

func GetRepoRef(r *http.Request) (string, error) {
	rawRef := chi.URLParam(r, RepoRefParamName)
	if rawRef == "" {
		return "", ErrRepoReferenceNotFound
	}

	// paths are unescaped and lower
	ref, err := url.PathUnescape(rawRef)
	return strings.ToLower(ref), err
}
