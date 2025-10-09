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

package secret

import (
	"encoding/json"
	"net/http"

	"github.com/harness/gitness/app/api/controller/secret"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/paths"
)

func HandleUpdate(secretCtrl *secret.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)

		in := new(secret.UpdateInput)
		err := json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequestf(ctx, w, "Invalid Request Body: %s.", err)
			return
		}

		secretRef, err := request.GetSecretRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}
		spaceRef, secretIdentifier, err := paths.DisectLeaf(secretRef)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
		}

		secret, err := secretCtrl.Update(ctx, session, spaceRef, secretIdentifier, in)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.JSON(w, http.StatusOK, secret.CopyWithoutData())
	}
}
