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

package upload

import (
	"net/http"

	"github.com/harness/gitness/app/api/controller/upload"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"

	"github.com/rs/zerolog/log"
)

func HandleDownoad(controller *upload.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		filename, err := request.GetRemainderFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		signedFileURL, file, err := controller.Download(ctx, session, repoRef, filename)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}
		if file != nil {
			render.Reader(ctx, w, http.StatusOK, file)
			err = file.Close()
			if err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("failed to close file after rendering")
			}
			return
		}
		http.Redirect(
			w,
			r,
			signedFileURL,
			http.StatusTemporaryRedirect,
		)
	}
}
