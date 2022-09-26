// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package users

import (
	"net/http"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/store"
)

// HandleFind returns an http.HandlerFunc that writes json-encoded
// user account information to the the response body.
func HandleFind(userStore store.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user, _ := request.UserFrom(ctx)

		render.JSON(w, http.StatusOK, user)
	}
}
