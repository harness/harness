// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"net/http"
	"strconv"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
)

/*
 * Required returns an http.HandlerFunc middleware that resolves the
 * space using the fqsn from the request and injects it into the request.
 * In case the fqsn isn't found or the space doesn't exist an error is rendered.
 */
func Required(spaces store.SpaceStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ref, err := request.GetSpaceRef(r)
			if err != nil {
				render.BadRequest(w, err)
				return
			}

			ctx := r.Context()
			var s *types.Space

			// check if ref is spaceId - ASSUMPTION: digit only is no valid space name
			id, err := strconv.ParseInt(ref, 10, 64)
			if err == nil {
				s, err = spaces.Find(ctx, id)
			} else {
				s, err = spaces.FindFqn(ctx, ref)
			}

			if err != nil {
				// TODO: what about errors that aren't notfound?
				render.NotFoundf(w, "Resolving space reference '%s' failed: %s", ref, err)
				return
			}

			next.ServeHTTP(w, r.WithContext(
				request.WithSpace(ctx, s),
			))
		})
	}
}
