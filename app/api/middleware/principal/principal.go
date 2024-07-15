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

package principal

import (
	"net/http"

	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

/*
 * RestrictTo returns an http.HandlerFunc middleware that ensures the principal
 * is of the provided type. In case there is no authenticated principal,
 * or the principal type doesn't match, an error is rendered.
 */
func RestrictTo(pType enum.PrincipalType) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			p, ok := request.PrincipalFrom(ctx)
			if !ok {
				log.Ctx(ctx).Debug().Msgf("Failed to get principal from session")

				render.Forbidden(ctx, w)
				return
			}
			if p.UID == types.AnonymousPrincipalUID {
				log.Ctx(ctx).Debug().Msgf("Valid principal is required, received an Anonymous.")

				// TODO: revert to Unauthorized once UI is handling it properly.
				render.NotFound(ctx, w)
				return
			}

			if p.Type != pType {
				log.Ctx(ctx).Debug().Msgf("Principal of type '%s' required.", pType)

				render.Forbidden(ctx, w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

/*
 * RestrictToAdmin returns an http.HandlerFunc middleware that ensures the principal
 * is an admin. In case there is no authenticated principal,
 * or the principal isn't an admin, an error is rendered.
 */
func RestrictToAdmin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			p, ok := request.PrincipalFrom(ctx)
			if !ok || !p.Admin {
				log.Ctx(ctx).Debug().Msg("No principal found or the principal is no admin")

				render.Forbidden(ctx, w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
