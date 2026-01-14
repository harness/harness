// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package account

import (
	"net/http"

	"github.com/harness/gitness/app/api/controller/user"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
)

// HandleLogout returns a http.HandlerFunc that deletes the
// user token being used in the respective request and logs the user out.
func HandleLogout(userCtrl *user.Controller, cookieName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		err := userCtrl.Logout(ctx, session)

		// best effort delete cookie even in case of errors, to avoid clients being unable to remove the cookie.
		// WARNING: It could be that the cookie is removed even though the token is still there in the DB.
		// However, we have APIs to list and delete session tokens, and expiry time is usually short.
		deleteTokenCookieIfPresent(r, w, cookieName)

		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.DeleteSuccessful(w)
	}
}
