package request

import (
	"net/http"
)

const (
	PathIDParamName = "pathId"
)

func GetPathID(r *http.Request) (int64, error) {
	return ParseAsInt64(r, PathIDParamName)
}
