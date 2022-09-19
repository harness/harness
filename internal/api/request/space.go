package request

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi"
)

const (
	SpaceRefParamName = "sref"
)

var (
	ErrSpaceReferenceNotFound = errors.New("no space reference found in request")
)

func GetSpaceRef(r *http.Request) (string, error) {
	rawRef := chi.URLParam(r, SpaceRefParamName)
	if rawRef == "" {
		return "", ErrSpaceReferenceNotFound
	}

	// paths are unescaped and lower
	ref, err := url.PathUnescape(rawRef)
	return strings.ToLower(ref), err
}
