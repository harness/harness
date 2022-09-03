// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package admin

import (
	"errors"
	"net/http"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

/*
 * Encorce returns an http.HandlerFunc middleware that authenticates
 * the http.Request and errors if the account cannot be authenticated.
 */
func Encorce(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user, ok := request.UserFrom(ctx)
		if !ok {
			render.Unauthorized(w, errors.New("Requires authentication"))
			return
		}

		if !user.Admin {
			render.Forbidden(w, errors.New("Forbidden"))
			return
		}

		next.ServeHTTP(w, r)
	})
}
