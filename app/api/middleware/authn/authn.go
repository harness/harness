// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package authn

import (
	"errors"
	"net/http"

	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authn"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

// Attempt returns an http.HandlerFunc middleware that authenticates
// the http.Request if authentication payload is available.
// Otherwise, an anonymous user session is used instead.
func Attempt(authenticator authn.Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := hlog.FromRequest(r)

			session, err := authenticator.Authenticate(r)
			if err != nil && !errors.Is(err, authn.ErrNoAuthData) {
				log.Debug().Err(err).Msg("authentication failed")

				render.Unauthorized(ctx, w)
				return
			}

			if errors.Is(err, authn.ErrNoAuthData) {
				log.Info().Msg("No authentication data found, continue as anonymous")
				session = &auth.Session{
					Principal: auth.AnonymousPrincipal,
				}
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
