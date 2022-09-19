// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/harness/gitness/internal/api/guard"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/hlog"
)

type repoCreateInput struct {
	Name        string `json:"name"`
	SpaceID     int64  `json:"spaceId"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	IsPublic    bool   `json:"isPublic"`
	ForkID      int64  `json:"forkId"`
}

/*
 * HandleCreate returns an http.HandlerFunc that creates a new repository.
 */
func HandleCreate(guard *guard.Guard, spaces store.SpaceStore, repos store.RepoStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := hlog.FromRequest(r)

		in := new(repoCreateInput)
		err := json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			log.Debug().Err(err).
				Msg("Decoding json body failed.")

			render.BadRequestf(w, "Invalid Request Body: %s.", err)
			return
		}

		// ensure we reference a space
		if in.SpaceID <= 0 {
			render.BadRequestf(w, "A repository can only be created within a space.")
			return
		}

		parentSpace, err := spaces.Find(ctx, in.SpaceID)
		if err != nil {
			log.Err(err).Msgf("Failed to get space with id '%d'.", in.SpaceID)

			render.UserfiedErrorOrInternal(w, err)
			return
		}

		// parentPath is assumed to be valid, in.Name gets validated in check.Repo function
		parentPath := parentSpace.Path

		/*
		 * AUTHORIZATION
		 * Create is a special case - check permission without specific resource
		 */
		scope := &types.Scope{SpacePath: parentPath}
		resource := &types.Resource{
			Type: enum.ResourceTypeRepo,
			Name: "",
		}
		if !guard.Enforce(w, r, scope, resource, enum.PermissionRepoCreate) {
			return
		}

		// get current user (safe to be there, or enforce would fail)
		usr, _ := request.UserFrom(ctx)

		// create new repo object
		repo := &types.Repository{
			Name:        strings.ToLower(in.Name),
			SpaceID:     in.SpaceID,
			DisplayName: in.DisplayName,
			Description: in.Description,
			IsPublic:    in.IsPublic,
			CreatedBy:   usr.ID,
			Created:     time.Now().UnixMilli(),
			Updated:     time.Now().UnixMilli(),
			ForkID:      in.ForkID,
		}

		// validate repo
		if err = check.Repo(repo); err != nil {
			render.UserfiedErrorOrInternal(w, err)
			return
		}

		// create in store
		err = repos.Create(ctx, repo)
		if err != nil {
			log.Error().Err(err).
				Msg("Repository creation failed.")

			render.UserfiedErrorOrInternal(w, err)
			return
		}

		render.JSON(w, http.StatusOK, repo)
	}
}
