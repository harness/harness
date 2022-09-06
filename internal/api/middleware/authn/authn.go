// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package authn

import (
	"net/http"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/auth/authn"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

/*
 * Attempt returns an http.HandlerFunc middleware that authenticates
 * the http.Request if authentication payload is available.
 */
func Attempt(authenticator authn.Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := authenticator.Authenticate(r)
			if err != nil {
				render.Unauthorized(w, err)
				return
			}

			// if there was no auth info - continue as is
			if user == nil {
				next.ServeHTTP(w, r)
				return
			}

			// otherwise update the logging context and inject user in context
			ctx := r.Context()
			log.Ctx(ctx).UpdateContext(func(c zerolog.Context) zerolog.Context {
				return c.Str("session_email", user.Email).Bool("session_admin", user.Admin)
			})

			next.ServeHTTP(w, r.WithContext(
				request.WithUser(ctx, user),
			))
		})
	}
}
