// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package account

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/user"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
)

// HandleLogout returns a http.HandlerFunc that deletes the
// user token being used in the respective request and logs the user out.
func HandleLogout(userCtrl *user.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		err := userCtrl.Logout(ctx, session)

		// best effort delete cookie even in case of errors, to avoid clients being unable to remove the cookie.
		// WARNING: It could be that the cookie is removed even though the token is still there in the DB.
		// However, we have APIs to list and delete session tokens, and expiry time is usually short.
		deleteTokenCookieIfPresent(r, w)

		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.DeleteSuccessful(w)
	}
}
