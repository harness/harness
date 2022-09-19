// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/harness/gitness/internal/api/guard"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/log"
)

type repoUpdateRequest struct {
	DisplayName *string `json:"displayName"`
	Description *string `json:"description"`
	IsPublic    *bool   `json:"isPublic"`
}

/*
 * Updates an existing repository.
 */
func HandleUpdate(guard *guard.Guard, repos store.RepoStore) http.HandlerFunc {
	return guard.Repo(
		enum.PermissionRepoEdit,
		false,
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			repo, _ := request.RepoFrom(ctx)

			in := new(repoUpdateRequest)
			err := json.NewDecoder(r.Body).Decode(in)
			if err != nil {
				render.BadRequestf(w, "Invalid request body: %s.", err)
				return
			}

			// update values only if provided
			if in.DisplayName != nil {
				repo.DisplayName = *in.DisplayName
			}
			if in.Description != nil {
				repo.Description = *in.Description
			}
			if in.IsPublic != nil {
				repo.IsPublic = *in.IsPublic
			}

			// always update time
			repo.Updated = time.Now().UnixMilli()

			// ensure provided values are valid
			if err = check.Repo(repo); err != nil {
				render.UserfiedErrorOrInternal(w, err)
				return
			}

			err = repos.Update(ctx, repo)
			if err != nil {
				log.Error().Err(err).Msg("Repository update failed.")

				render.UserfiedErrorOrInternal(w, err)
				return
			}

			render.JSON(w, http.StatusOK, repo)
		})
}
