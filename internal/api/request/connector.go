package request

import (
	"net/http"
	"net/url"
)

const (
	PathParamConnectorRef = "connector_ref"
)

func GetConnectorRefFromPath(r *http.Request) (string, error) {
	rawRef, err := PathParamOrError(r, PathParamConnectorRef)
	if err != nil {
		return "", err
	}

	// paths are unescaped
	return url.PathUnescape(rawRef)
}
