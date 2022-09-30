// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package resolve

import (
	"net/http"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/store"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

/*
 * ServiceAccount returns an http.HandlerFunc middleware that resolves the
 * service account using the id from the request and injects it into the request.
 * In case the id isn't found or the service account doesn't exist an error is rendered.
 */
func ServiceAccount(saStore store.ServiceAccountStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := hlog.FromRequest(r)

			uid, err := request.GetServiceAccountUID(r)
			if err != nil {
				log.Info().Err(err).Msgf("Receieved no or invalid service account uid")

				render.BadRequest(w)
				return
			}

			sa, err := saStore.FindUID(ctx, uid)
			if err != nil {
				log.Warn().Err(err).Msgf("Failed to get service account with uid '%s'.", uid)

				render.UserfiedErrorOrInternal(w, err)
				return
			}

			// Update the logging context and inject repo in context
			log.UpdateContext(func(c zerolog.Context) zerolog.Context {
				return c.Str("sa_uid", sa.UID)
			})

			next.ServeHTTP(w, r.WithContext(
				request.WithServiceAccount(ctx, sa),
			))
		})
	}
}
