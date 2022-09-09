package request

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"
)

const (
	PathIdParamName = "pathId"
)

func GetPathId(r *http.Request) (int64, error) {
	rawId := chi.URLParam(r, PathIdParamName)
	if rawId == "" {
		return 0, errors.New("Path id parameter not found in request.")
	}

	id, err := strconv.ParseInt(rawId, 10, 64)
	if err != nil {
		return 0, err
	}

	return id, nil
}
