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

func GetSpaceRef(r *http.Request) (string, error) {
	rawRef := chi.URLParam(r, SpaceRefParamName)
	if rawRef == "" {
		return "", errors.New("Space ref parameter not found in request.")
	}

	// fqns are unescaped and lower
	ref, err := url.PathUnescape(rawRef)
	return strings.ToLower(ref), err
}
