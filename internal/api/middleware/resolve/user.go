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
 * User returns an http.HandlerFunc middleware that resolves the
 * user using the id from the request and injects it into the request.
 * In case the id isn't found or the service account doesn't exist an error is rendered.
 */
func User(userStore store.UserStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := hlog.FromRequest(r)

			id, err := request.GetUserID(r)
			if err != nil {
				log.Info().Err(err).Msgf("Receieved no or invalid user id")

				render.BadRequest(w)
				return
			}

			user, err := userStore.Find(ctx, id)
			if err != nil {
				log.Info().Err(err).Msgf("Failed to get user with id '%d'.", id)

				render.UserfiedErrorOrInternal(w, err)
				return
			}

			// Update the logging context and inject repo in context
			log.UpdateContext(func(c zerolog.Context) zerolog.Context {
				return c.Int64("user_id", user.ID).Str("user_name", user.Name)
			})

			next.ServeHTTP(w, r.WithContext(
				request.WithUser(ctx, user),
			))
		})
	}
}
