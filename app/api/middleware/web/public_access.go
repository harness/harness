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

package web

import (
	"errors"
	"net/http"

	"github.com/harness/gitness/app/auth/authn"
	"github.com/harness/gitness/web"

	"github.com/rs/zerolog/hlog"
)

// PublicAccess enables rendering of the UI in public access mode if
// public access is enabled in the configuration and the request contains no logged-in user.
func PublicAccess(
	publicAccessEnabled bool,
	authenticator authn.Authenticator,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if !publicAccessEnabled {
			return next
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// slightly more expensive to authenticate the user, but we can't make assumptions about the authenticator.
			_, err := authenticator.Authenticate(r)
			if errors.Is(err, authn.ErrNoAuthData) {
				r = r.WithContext(web.WithRenderPublicAccess(r.Context(), true))
			} else if err != nil {
				// in case of failure to authenticate the user treat it as attempt to login
				hlog.FromRequest(r).Warn().Err(err).Msgf(
					"failed to authenticate principal, continue without public access UI",
				)
			}

			next.ServeHTTP(w, r)
		})
	}
}
