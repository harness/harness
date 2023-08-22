package request

import (
	"net/http"
	"net/url"
)

const (
	PathParamTemplateRef = "template_ref"
)

func GetTemplateRefFromPath(r *http.Request) (string, error) {
	rawRef, err := PathParamOrError(r, PathParamTemplateRef)
	if err != nil {
		return "", err
	}

	// paths are unescaped
	return url.PathUnescape(rawRef)
}
