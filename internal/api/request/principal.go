package request

import (
	"net/http"
)

const (
	UserIDParamName           = "userId"
	ServiceAccountIDParamName = "saId"
)

func GetUserID(r *http.Request) (int64, error) {
	return ParseAsInt64(r, UserIDParamName)
}

func GetServiceAccountID(r *http.Request) (int64, error) {
	return ParseAsInt64(r, ServiceAccountIDParamName)
}
