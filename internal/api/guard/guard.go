// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package guard

import (
	"fmt"
	"net/http"

	"github.com/rs/zerolog/hlog"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
	"github.com/pkg/errors"
)

var (
	ErrNotAuthenticated = errors.New("not authenticated")
	ErrNotAuthorized    = errors.New("not authorized")
)

type Guard struct {
	authorizer authz.Authorizer
}

func New(authorizer authz.Authorizer) *Guard {
	return &Guard{authorizer: authorizer}
}

// EnforceAdmin is a middleware that enforces that the user is authenticated and an admin.
func (g *Guard) EnforceAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user, ok := request.UserFrom(ctx)
		if !ok {
			render.Unauthorized(w)
			return
		}

		if !user.Admin {
			render.Forbidden(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// EnforceAuthenticated is a middleware that enforces that the user is authenticated.
func (g *Guard) EnforceAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		_, ok := request.UserFrom(ctx)
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
		render.ErrorObject(w, http.StatusUnauthorized, render.ErrUnauthorized)
	case errors.Is(err, ErrNotAuthorized):
		// log error for debugging.
		hlog.FromRequest(r).Debug().Msgf("User not authorized to perform %s on resource %v in scope %v",
			permission,
			resource,
			scope)

		render.Forbidden(w)
	case err != nil:
		// log err for debugging
		hlog.FromRequest(r).Err(err).Msg("Encountered unexpected error while enforcing permission.")
		render.InternalError(w)
	}
	return err == nil
}

// Check whether the principal executing the request has the requested permission on the resource.
// Returns nil if the user is confirmed to be permitted to execute the action, otherwise returns errors
// NotAuthenticated, NotAuthorized, or any unerlaying error.
func (g *Guard) Check(r *http.Request, scope *types.Scope, resource *types.Resource, permission enum.Permission) error {
	u, present := request.UserFrom(r.Context())
	if !present {
		return ErrNotAuthenticated
	}

	// TODO: don't hardcode principal type USER
	authorized, err := g.authorizer.Check(
		r.Context(),
		enum.PrincipalTypeUser,
		fmt.Sprint(u.ID),
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
