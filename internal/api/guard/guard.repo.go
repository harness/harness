// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package guard

import (
	"net/http"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/paths"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/hlog"
)

/*
 * Returns a middleware that guards repository related handlers from being executed.
 * Only principals that are authorized are able to execute the handler, everyone else is forbidden,
 * unless orPublic is configured and the repository is public.
 *
 * Assumes the repository is already available in the request context.
 */
func (e *Guard) ForRepo(requiredPermission enum.Permission, orPublic bool) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return e.Repo(requiredPermission, orPublic, h.ServeHTTP)
	}
}

/*
 * Returns an http.HandlerFunc that guards a repository related http.HandlerFunc from being executed.
 * Only principals that are authorized are able to execute the handler, everyone else is forbidden,
 * unless orPublic is configured and the repository is public.
 *
 * Assumes the repository is already available in the request context.
 */
func (g *Guard) Repo(permission enum.Permission, orPublic bool, guarded http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		rep, ok := request.RepoFrom(ctx)
		if !ok {
			// log error for debugging
			hlog.FromRequest(r).Error().Msg("Method expects the repository to be availabe in the request context, but it wasnt.")

			render.InternalError(w)
			return
		}

		// Enforce permission (renders error)
		if !(orPublic && rep.IsPublic) && !g.EnforceRepo(w, r, permission, rep.Path) {
			return
		}

		// executed guarded function
		guarded(w, r)
	}
}

/*
 * Enforces that the executing principal has requested permission on the repository.
 * Returns true if that is the case, otherwise renders the appropriate error and returns false.
 */
func (g *Guard) EnforceRepo(w http.ResponseWriter, r *http.Request, permission enum.Permission, path string) bool {
	spacePath, name, err := paths.Disect(path)
	if err != nil {
		// log error for debugging
		hlog.FromRequest(r).Err(err).Msgf("Failed to disect path '%s'.", path)

		render.InternalError(w)
		return false
	}

	scope := &types.Scope{SpacePath: spacePath}
	resource := &types.Resource{
		Type: enum.ResourceTypeRepo,
		Name: name,
	}

	return g.Enforce(w, r, scope, resource, permission)
}

/*
 * Checks whether the principal executing the request has the requested permission on the repository.
 * Returns nil if the user is confirmed to be permitted to execute the action, otherwise returns errors
 * NotAuthenticated, NotAuthorized, or any unerlaying error.
 */
func (g *Guard) CheckRepo(r *http.Request, permission enum.Permission, path string) error {
	parentSpace, name, err := paths.Disect(path)
	if err != nil {
		return errors.Wrapf(err, "Failed to disect path '%s'", path)
	}

	scope := &types.Scope{SpacePath: parentSpace}
	resource := &types.Resource{
		Type: enum.ResourceTypeRepo,
		Name: name,
	}

	return g.Check(r, scope, resource, permission)
}
