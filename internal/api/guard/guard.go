// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package guard

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/api/space"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type Guard struct {
	authorizer authz.Authorizer
	spaces     store.SpaceStore
}

func New(spaces store.SpaceStore, authorizer authz.Authorizer) *Guard {
	return &Guard{authorizer: authorizer, spaces: spaces}
}

/*
 * Returns a middleware that guards space related handlers from being executed.
 * Only principals that are authorized are able to execute the handler, everyone else is forbidden.
 */
func (e *Guard) ForSpace(requiredPermission enum.Permission) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return e.Space(requiredPermission, h.ServeHTTP)
	}
}

/*
 * Returns an http.HandlerFunc that guards a space related http.HandlerFunc from being executed.
 * Only principals that are authorized are able to execute the handler, everyone else is forbidden.
 */
func (e *Guard) Space(requiredPermission enum.Permission, guarded http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, _, err := space.UsingRefParam(r, e.spaces)
		if err != nil {
			render.Forbidden(w, err)
			return
		}

		u, present := request.UserFrom(r.Context())
		if !present {
			render.InternalError(w, errors.New("Unexpectetly didn't find a user."))
			return
		}

		err = e.authorizer.Check(
			enum.PrincipalTypeUser,
			fmt.Sprint(u.ID),
			types.Resource{
				Type:       enum.ResourceTypeSpace,
				Identifier: s.Fqsn,
			},
			requiredPermission)
		if err != nil {
			render.Forbidden(w, err)
			return
		}

		guarded(w, r)
	}
}

/*
 * Checks whether the principal executing the request has the requested permission on the space.
 * Returns true if the user is permitted to execute the action, otherwise renders an error and returns false.
 */
func (e *Guard) CheckSpace(w http.ResponseWriter, r *http.Request, requiredPermission enum.Permission, fqsn string) bool {
	u, present := request.UserFrom(r.Context())
	if !present {
		render.InternalError(w, errors.New("Unexpectetly didn't find a user."))
		return false
	}

	err := e.authorizer.Check(
		enum.PrincipalTypeUser,
		fmt.Sprint(u.ID),
		types.Resource{
			Type:       enum.ResourceTypeSpace,
			Identifier: fqsn,
		},
		requiredPermission)
	if err != nil {
		render.Forbidden(w, err)
		return false
	}

	return true
}
