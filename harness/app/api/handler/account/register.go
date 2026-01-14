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
	"encoding/json"
	"net/http"

	"github.com/harness/gitness/app/api/controller/system"
	"github.com/harness/gitness/app/api/controller/user"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
)

// HandleRegister returns an http.HandlerFunc that processes an http.Request
// to register the named user account with the system.
func HandleRegister(userCtrl *user.Controller, sysCtrl *system.Controller, cookieName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		includeCookie, err := request.GetIncludeCookieFromQueryOrDefault(r, false)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		in := new(user.RegisterInput)
		err = json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequestf(ctx, w, "Invalid request body: %s.", err)
			return
		}

		tokenResponse, err := userCtrl.Register(ctx, sysCtrl, in)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		if includeCookie {
			includeTokenCookie(r, w, tokenResponse, cookieName)
		}

		render.JSON(w, http.StatusOK, tokenResponse)
	}
}
