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

package repo

import (
	"fmt"
	"net/http"

	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/api/render"
	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/git/api"
)

func HandleArchive(repoCtrl *repo.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		session, _ := request.AuthSessionFrom(ctx)
		repoRef, err := request.GetRepoRefFromPath(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		params, filename, err := request.ParseArchiveParams(r)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}

		var contentType string
		switch params.Format {
		case api.ArchiveFormatTar:
			contentType = "application/tar"
		case api.ArchiveFormatZip:
			contentType = "application/zip"
		case api.ArchiveFormatTarGz, api.ArchiveFormatTgz:
			contentType = "application/gzip"
		default:
			contentType = "application/zip"
		}

		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		w.Header().Set("Content-Type", contentType)

		err = repoCtrl.Archive(ctx, session, repoRef, params, w)
		if err != nil {
			render.TranslatedUserError(ctx, w, err)
			return
		}
	}
}
