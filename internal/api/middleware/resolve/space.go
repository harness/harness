// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package resolve

import (
	"net/http"
	"strconv"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

/*
 * Space returns an http.HandlerFunc middleware that resolves the
 * space using the fqsn from the request and injects it into the request.
 * In case the fqsn isn't found or the space doesn't exist an error is rendered.
 */
func Space(spaceStore store.SpaceStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := hlog.FromRequest(r)

			ref, err := request.GetSpaceRef(r)
			if err != nil {
				log.Info().Err(err).Msgf("Receieved no or invalid space ref")

				render.BadRequest(w)
				return
			}

			var space *types.Space

			// check if ref is spaceId - ASSUMPTION: digit only is no valid space name
			id, err := strconv.ParseInt(ref, 10, 64)
			if err == nil {
				space, err = spaceStore.Find(ctx, id)
			} else {
				space, err = spaceStore.FindByPath(ctx, ref)
			}

			if err != nil {
				log.Debug().Err(err).Msgf("Failed to get space using ref '%s'.", ref)

				render.UserfiedErrorOrInternal(w, err)
				return
			}

			// Update the logging context and inject repo in context
			log.UpdateContext(func(c zerolog.Context) zerolog.Context {
				return c.Int64("space_id", space.ID).Str("space_path", space.Path)
			})

			next.ServeHTTP(w, r.WithContext(
				request.WithSpace(ctx, space),
			))
		})
	}
}
