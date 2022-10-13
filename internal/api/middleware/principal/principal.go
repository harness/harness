// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package principal

import (
	"net/http"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/hlog"
)

/*
 * UserFromPrincipal returns an http.HandlerFunc middleware that ensures the principal
 * is a user and injects it into the request. In case the princicipal isn't of type user,
 * or the user doesn't exist, an error is rendered.
 */
func RestrictTo(pType enum.PrincipalType) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := hlog.FromRequest(r)

			p, ok := request.PrincipalFrom(ctx)
			if !ok || p.Type != pType {
				log.Debug().Msgf("Principal of type '%s' required.", pType)

				render.Forbidden(w)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
