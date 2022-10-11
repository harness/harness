package request

import (
	"net/http"
)

const (
	PathParamTokenID = "tokenID"
)

func GetTokenIDFromPath(r *http.Request) (int64, error) {
	return PathParamAsInt64(r, PathParamTokenID)
}
