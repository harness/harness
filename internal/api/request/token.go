package request

import (
	"net/http"
)

const (
	PathParamTokenID = "tokenID"
)

func GetTokenID(r *http.Request) (int64, error) {
	return ParseAsInt64(r, PathParamTokenID)
}
