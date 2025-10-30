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

package space

import (
	"net/http"

	"github.com/harness/gitness/app/api/controller/space"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
)

// HandleMembershipDelete handles API that deletes an existing space membership.
func HandleMembershipDelete(spaceCtrl *space.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		spaceRef, err := request.GetSpaceRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		userUID, err := request.GetUserUIDFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		err = spaceCtrl.MembershipDelete(ctx, session, spaceRef, userUID)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.DeleteSuccessful(w)
	}
}
