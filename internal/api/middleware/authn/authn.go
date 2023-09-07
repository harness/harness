// Copyright 2022 Harness Inc. All rights reserved.
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
func Attempt(authenticator authn.Authenticator, sourceRouter authn.SourceRouter) func(http.Handler) http.Handler {
	return performAuthentication(authenticator, false, sourceRouter)
}

// Required returns an http.HandlerFunc middleware that authenticates
// the http.Request and fails the request if no auth data was available.
func Required(authenticator authn.Authenticator, sourceRouter authn.SourceRouter) func(http.Handler) http.Handler {
	return performAuthentication(authenticator, true, sourceRouter)
}

// performAuthentication returns an http.HandlerFunc middleware that authenticates
// the http.Request if authentication payload is available.
// Depending on whether it is required or not, the request will be failed.
func performAuthentication(
	authenticator authn.Authenticator,
	required bool, sourceRouter authn.SourceRouter,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := hlog.FromRequest(r)

			session, err := authenticator.Authenticate(r, sourceRouter)
			if err != nil {
				if !errors.Is(err, authn.ErrNoAuthData) {
					// log error to help with investigating any auth related errors
					log.Warn().Err(err).Msg("authentication failed")
				}

				if required {
					render.Unauthorized(w)
					return
				}

				// if there was no (valid) auth data in the request, then continue without session
				next.ServeHTTP(w, r)
				return
			}

			if session == nil {
				// when err == nil session should never be nil!
				log.Error().Msg("auth session is nil eventhough the authenticator didn't return any error!")

				render.InternalError(w)
				return
			}

			// Update the logging context and inject principal in context
			log.UpdateContext(func(c zerolog.Context) zerolog.Context {
				return c.
					Str("principal_uid", session.Principal.UID).
					Str("principal_type", string(session.Principal.Type)).
					Bool("principal_admin", session.Principal.Admin)
			})

			next.ServeHTTP(w, r.WithContext(
				request.WithAuthSession(ctx, session),
			))
		})
	}
}
