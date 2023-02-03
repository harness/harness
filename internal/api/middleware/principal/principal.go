// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package principal

import (
	"net/http"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
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
			if !ok || p.Type != pType {
				log.Ctx(ctx).Debug().Msgf("Principal of type '%s' required.", pType)

				render.Forbidden(w)
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

				render.Forbidden(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
