// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package access

import (
	"errors"
	"net/http"

	"github.com/harness/scm/internal/api/render"
	"github.com/harness/scm/internal/api/request"
)

// SystemAdmin returns an http.HandlerFunc middleware that authorizes
// the user access to system administration capabilities.
func SystemAdmin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			user, ok := request.UserFrom(ctx)
			if !ok {
				render.ErrorCode(w, errors.New("Requires authentication"), 401)
				return
			}
			if !user.Admin {
				render.ErrorCode(w, errors.New("Forbidden"), 403)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
