// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package resolve

import (
	"net/http"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/hlog"
)

/*
 * UserFromPrincipal returns an http.HandlerFunc middleware that ensures the principal
 * is a user and injects it into the request. In case the princicipal isn't of type user,
 * or the user doesn't exist, an error is rendered.
 */
func UserFromPrincipal(userStore store.UserStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := hlog.FromRequest(r)

			p, ok := request.PrincipalFrom(ctx)
			if !ok || p.Type != enum.PrincipalTypeUser {
				log.Warn().Msgf("Principal of type user required.")

				render.Forbidden(w)
				return
			}

			user, err := userStore.Find(ctx, p.ID)
			if err != nil {
				// not expected to happen, as user was retrieved to get principal in the first place
				// Only legitimate way is a racing condition with a user delete - otherwise it's a bug
				log.Warn().Err(err).Msgf("Unable to find user with id '%d'", p.ID)

				render.InternalError(w)
				return
			}

			next.ServeHTTP(w, r.WithContext(
				request.WithUser(ctx, user),
			))
		})
	}
}
