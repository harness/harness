// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package guard

import (
	"net/http"

	"github.com/rs/zerolog/hlog"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/paths"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
	"github.com/pkg/errors"
)

var (
	ErrNotAuthenticated          = errors.New("not authenticated")
	ErrNotAuthorized             = errors.New("not authorized")
	ErrParentResourceTypeUnknown = errors.New("Unknown parent resource type")
)

type Guard struct {
	authorizer authz.Authorizer
	spaceStore store.SpaceStore
	repoStore  store.RepoStore
}

func New(authorizer authz.Authorizer, spaceStore store.SpaceStore, repoStore store.RepoStore) *Guard {
	return &Guard{authorizer, spaceStore, repoStore}
}

// EnforceAdmin is a middleware that enforces that the principal is authenticated and an admin.
func (g *Guard) EnforceAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		p, ok := request.PrincipalFrom(ctx)
		if !ok {
			render.Unauthorized(w)
			return
		}

		if !p.Admin {
			render.Forbidden(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// EnforceAuthenticated is a middleware that enforces that the principal is authenticated.
func (g *Guard) EnforceAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		_, ok := request.AuthSessionFrom(ctx)
		if !ok {
			render.Unauthorized(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Enforce that the executing principal has requested permission on the resource
// returns true if it's the case, otherwise renders the appropriate error and returns false.
func (g *Guard) Enforce(w http.ResponseWriter, r *http.Request, scope *types.Scope, resource *types.Resource,
	permission enum.Permission) bool {
	err := g.Check(r, scope, resource, permission)
	// render error if needed
	switch {
	case errors.Is(err, ErrNotAuthenticated):
		render.Unauthorized(w)
	case errors.Is(err, ErrNotAuthorized):
		render.Forbidden(w)
	case err != nil:
		// log err for debugging
		hlog.FromRequest(r).Err(err).Msg("Encountered unexpected error while enforcing permission.")
		render.InternalError(w)
	}
	return err == nil
}

// Check whether the principal executing the request has the requested permission on the resource.
// Returns nil if the principal is confirmed to be permitted to execute the action, otherwise returns errors
// NotAuthenticated, NotAuthorized, or any unerlaying error.
func (g *Guard) Check(r *http.Request, scope *types.Scope, resource *types.Resource, permission enum.Permission) error {
	session, present := request.AuthSessionFrom(r.Context())
	if !present {
		return ErrNotAuthenticated
	}

	authorized, err := g.authorizer.Check(
		r.Context(),
		session,
		scope,
		resource,
		permission)
	if err != nil {
		return err
	}

	if !authorized {
		return ErrNotAuthorized
	}

	return nil
}

/*
 * EnforceInParentScope enforces that the executing principal has requested permission on
 * the specificed resource in the scope of the parent.
 * Returns true if that is the case, otherwise renders the appropriate error and returns false.
 */
func (g *Guard) EnforceInParentScope(w http.ResponseWriter, r *http.Request, resource *types.Resource,
	permission enum.Permission, parentType enum.ParentResourceType, parentID int64) bool {
	scope, err := g.getScopeForParent(r, parentType, parentID)
	if err != nil {
		render.UserfiedErrorOrInternal(w, err)
		return false
	}

	return g.Enforce(w, r, scope, resource, permission)
}

/*
 * CheckInParentScope checks if the executing principal has requested permission on
 * the specificed resource in the scope of the parent.
 * Returns nil if the principal is confirmed to be permitted to execute the action, otherwise returns errors
 * NotAuthenticated, NotAuthorized, or any unerlaying error.
 */
func (g *Guard) CheckInParentScope(r *http.Request, resource *types.Resource,
	permission enum.Permission, parentType enum.ParentResourceType, parentID int64) error {
	scope, err := g.getScopeForParent(r, parentType, parentID)
	if err != nil {
		return err
	}

	return g.Check(r, scope, resource, permission)
}

// getScopeForParent Returns the scope for a given resource parent (space or repo).
func (g *Guard) getScopeForParent(r *http.Request,
	parentType enum.ParentResourceType, parentID int64) (*types.Scope, error) {
	ctx := r.Context()
	log := hlog.FromRequest(r)

	// TODO: Can this be done cleaner?
	switch parentType {
	case enum.ParentResourceTypeSpace:
		space, err := g.spaceStore.Find(ctx, parentID)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to find space")

			return nil, err
		}

		return &types.Scope{SpacePath: space.Path}, nil

	case enum.ParentResourceTypeRepo:
		repo, err := g.repoStore.Find(ctx, parentID)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to find repo")

			return nil, err
		}

		spacePath, repoName, err := paths.Disect(repo.Path)
		if err != nil {
			log.Error().Err(err).Msg("Failed to disect path")

			return nil, err
		}

		return &types.Scope{SpacePath: spacePath, Repo: repoName}, nil

	default:
		log.Error().Msgf("Unsupported parent type encountered: '%s'", parentType)

		return nil, ErrParentResourceTypeUnknown
	}
}
