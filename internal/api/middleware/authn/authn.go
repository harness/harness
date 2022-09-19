// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package authn

import (
	"errors"
	"net/http"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/auth/authn"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

// Attempt returns an http.HandlerFunc middleware that authenticates
// the http.Request if authentication payload is available.
func Attempt(authenticator authn.Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := hlog.FromRequest(r)

			user, err := authenticator.Authenticate(r)

			if errors.Is(err, authn.ErrNoAuthData) {
				// if there was no auth data in the request - continue as is
				next.ServeHTTP(w, r)
				return
			}

			if err != nil {
				// for any other error we fail
				render.Unauthorized(w)
				return
			}

			if user == nil {
				// when err == nil user should never be nil!
				log.Error().Msg("User is nil eventhough the authenticator didn't return any error!")

				render.InternalError(w)
				return
			}

			// Update the logging context and inject user in context
			log.UpdateContext(func(c zerolog.Context) zerolog.Context {
				return c.Int64("user_id", user.ID).Bool("user_admin", user.Admin)
			})

			next.ServeHTTP(w, r.WithContext(
				request.WithUser(ctx, user),
			))
		})
	}
}
