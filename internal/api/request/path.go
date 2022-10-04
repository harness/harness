package request

import (
	"net/http"
)

const (
	PathParamPathID = "pathID"
)

func GetPathID(r *http.Request) (int64, error) {
	return ParseAsInt64(r, PathParamPathID)
}
