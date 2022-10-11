package request

import (
	"net/http"
)

const (
	PathParamPathID = "pathID"
)

func GetPathIDFromPath(r *http.Request) (int64, error) {
	return PathParamAsInt64(r, PathParamPathID)
}
