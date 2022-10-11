package request

import (
	"net/http"
	"net/url"
	"strings"
)

const (
	PathParamRepoRef = "repositoryRef"
)

func GetRepoRefFromPath(r *http.Request) (string, error) {
	rawRef, err := PathParamOrError(r, PathParamRepoRef)
	if err != nil {
		return "", err
	}

	// paths are unescaped and lower
	ref, err := url.PathUnescape(rawRef)
	return strings.ToLower(ref), err
}
