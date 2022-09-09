package request

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi"
	"github.com/harness/gitness/types/errs"
)

const (
	RepoRefParamName = "rref"
)

func GetRepoRef(r *http.Request) (string, error) {
	rawRef := chi.URLParam(r, RepoRefParamName)
	if rawRef == "" {
		return "", errs.RepoReferenceNotFoundInRequest
	}

	// paths are unescaped and lower
	ref, err := url.PathUnescape(rawRef)
	return strings.ToLower(ref), err
}
