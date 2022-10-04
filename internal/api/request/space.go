package request

import (
	"net/http"
	"net/url"
	"strings"
)

const (
	PathParamSpaceRef = "spaceRef"
)

func GetSpaceRef(r *http.Request) (string, error) {
	rawRef, err := ParamOrError(r, PathParamSpaceRef)
	if err != nil {
		return "", err
	}

	// paths are unescaped and lower
	ref, err := url.PathUnescape(rawRef)
	return strings.ToLower(ref), err
}
