package request

import (
	"net/http"
)

const (
	UserUIDParamName           = "userUID"
	ServiceAccountUIDParamName = "saUID"
)

func GetUserUID(r *http.Request) (string, error) {
	return ParamOrError(r, UserUIDParamName)
}

func GetServiceAccountUID(r *http.Request) (string, error) {
	return ParamOrError(r, ServiceAccountUIDParamName)
}
