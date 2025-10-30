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

package resource

import (
	"net/http"

	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/resources"

	"github.com/rs/zerolog/log"
)

func HandleGitIgnores() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		files, err := resources.GitIgnores()
		if err != nil {
			log.Ctx(ctx).Err(err).Msgf("Failed to load gitignore files")
			render.InternalError(ctx, w)
			return
		}
		render.JSON(w, http.StatusOK, files)
	}
}

func HandleLicences() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		response, err := resources.Licenses()
		if err != nil {
			log.Ctx(ctx).Err(err).Msgf("Failed to load license files")
			render.InternalError(ctx, w)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(response)
	}
}
