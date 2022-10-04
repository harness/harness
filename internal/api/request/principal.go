package request

import (
	"net/http"
)

const (
	PathParamUserUID           = "userUID"
	PathParamServiceAccountUID = "saUID"
)

func GetUserUID(r *http.Request) (string, error) {
	return ParamOrError(r, PathParamUserUID)
}

func GetServiceAccountUID(r *http.Request) (string, error) {
	return ParamOrError(r, PathParamServiceAccountUID)
}
