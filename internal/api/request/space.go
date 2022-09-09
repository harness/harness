package request

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi"
	"github.com/harness/gitness/internal/errs"
)

const (
	SpaceRefParamName = "sref"
)

func GetSpaceRef(r *http.Request) (string, error) {
	rawRef := chi.URLParam(r, SpaceRefParamName)
	if rawRef == "" {
		return "", errs.SpaceReferenceNotFoundInRequest
	}

	// paths are unescaped and lower
	ref, err := url.PathUnescape(rawRef)
	return strings.ToLower(ref), err
}
