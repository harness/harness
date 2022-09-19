package request

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
)

const (
	PathIDParamName = "pathId"
)

func GetPathID(r *http.Request) (int64, error) {
	rawID := chi.URLParam(r, PathIDParamName)
	if rawID == "" {
		return 0, errors.New("path id parameter not found in request")
	}

	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		return 0, err
	}

	return id, nil
}
