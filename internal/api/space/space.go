package space

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
)

const (
	RefParamName = "sref"
)

/*
 * Tries to get the space using the sref matched during routing.
 * Always returns the sref.
 */
func UsingRefParam(r *http.Request, spaces store.SpaceStore) (*types.Space, string, error) {

	sref, err := GetRefParam(r)
	if err != nil {
		return nil, "", err
	} else if sref == "" {
		return nil, "", errors.New("Parameter not found")
	}

	// check if sref is spaceId - ASSUMPTION: digit only is no valid space name
	if id, err := strconv.ParseInt(sref, 10, 64); err == nil {
		s, err := spaces.Find(r.Context(), id)
		return s, sref, err
	}

	// else assume its fqsn
	s, err := spaces.FindFqsn(r.Context(), sref)
	return s, sref, err
}

func GetRefParam(r *http.Request) (string, error) {
	rawSref := chi.URLParam(r, RefParamName)
	// always unescape - if it's the Id it'll be a noop
	return url.PathUnescape(rawSref)
}
