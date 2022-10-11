package request

import (
	"net/http"
)

const (
	PathParamUserUID           = "userUID"
	PathParamServiceAccountUID = "saUID"
)

func GetUserUIDFromPath(r *http.Request) (string, error) {
	return PathParamOrError(r, PathParamUserUID)
}

func GetServiceAccountUIDFromPath(r *http.Request) (string, error) {
	return PathParamOrError(r, PathParamServiceAccountUID)
}
