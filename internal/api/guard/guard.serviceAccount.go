// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package guard

import (
	"net/http"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/hlog"
)

/*
 * Returns a middleware that guards service account related handlers from being executed.
 * Only principals that are authorized are able to execute the handler, everyone else is forbidden.
 *
 * Assumes the service account is already available in the request context.
 */
func (g *Guard) ForServiceAccount(requiredPermission enum.Permission) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return g.ServiceAccount(requiredPermission, h.ServeHTTP)
	}
}

/*
 * Returns an http.HandlerFunc that guards service account related http.HandlerFunc from being executed.
 * Only principals that are authorized are able to execute the handler, everyone else is forbidden.
 * Assumes the service account is already available in the request context.
 */
func (g *Guard) ServiceAccount(permission enum.Permission, guarded http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := hlog.FromRequest(r)

		sa, ok := request.ServiceAccountFrom(ctx)
		if !ok {
			log.Error().Msg("Method expects the service account in request context, but wasnt.")

			render.InternalError(w)
			return
		}

		// Enforce permission (renders error)
		// TODO: Currently we don't support per service account RBAC (only all or nothing)
		if !g.EnforceInParentScope(w, r, &types.Resource{
			Type: enum.ResourceTypeServiceAccount,
			Name: ""},
			permission, sa.ParentType, sa.ParentID) {
			return
		}

		// executed guarded function
		guarded(w, r)
	}
}
