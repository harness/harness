package request

import (
	"net/http"
)

const (
	PatIDParamName          = "patId"
	SatIDParamName          = "satId"
	SessionTokenIDParamName = "sessionTokenId"
)

func GetPatID(r *http.Request) (int64, error) {
	return ParseAsInt64(r, PatIDParamName)
}

func GetSatID(r *http.Request) (int64, error) {
	return ParseAsInt64(r, SatIDParamName)
}

func GetSessionTokenID(r *http.Request) (int64, error) {
	return ParseAsInt64(r, SessionTokenIDParamName)
}
