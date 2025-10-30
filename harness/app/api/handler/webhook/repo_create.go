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

package webhook

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/harness/gitness/app/api/controller/webhook"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/types"
)

// HandleCreateRepo returns a http.HandlerFunc that creates a new webhook.
func HandleCreateRepo(webhookCtrl *webhook.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			render.BadRequestf(ctx, w, "Invalid Request Body: %s.", err)
			return
		}

		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		in := new(types.WebhookCreateInput)
		readerCloser := io.NopCloser(bytes.NewReader(bodyBytes))
		err = json.NewDecoder(readerCloser).Decode(in)
		if err != nil {
			render.BadRequestf(ctx, w, "Invalid Request Body: %s.", err)
			return
		}

		var signatureData *types.WebhookSignatureMetadata
		signature := request.GetSignatureFromHeaderOrDefault(r, "")
		if signature != "" {
			signatureData = new(types.WebhookSignatureMetadata)
			signatureData.Signature = signature
			signatureData.BodyBytes = bodyBytes
		}

		hook, err := webhookCtrl.CreateRepo(ctx, session, repoRef, in, signatureData)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		render.JSON(w, http.StatusCreated, hook)
	}
}
