package request

import (
	"net/http"
	"net/url"
	"strings"
)

const (
	PathParamSpaceRef = "spaceRef"
)

func GetSpaceRefFromPath(r *http.Request) (string, error) {
	rawRef, err := PathParamOrError(r, PathParamSpaceRef)
	if err != nil {
		return "", err
	}

	// paths are unescaped and lower
	ref, err := url.PathUnescape(rawRef)
	return strings.ToLower(ref), err
}
