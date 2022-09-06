// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"encoding/json"
	"errors"
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
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	IsPublic    bool   `json:"isPublic"`
	ForkId      int64  `json:"forkId"`
}

/*
 * HandleCreate returns an http.HandlerFunc that creates a new repository.
 */
func HandleCreate(guard *guard.Guard, spaces store.SpaceStore, repos store.RepoStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := hlog.FromRequest(r)

		// get fqn (requires parent of repo)
		rref, err := request.GetRepoRef(r)
		if err != nil {
			render.BadRequest(w, err)
			log.Debug().
				Err(err).
				Msgf("Failed to get the fqn.")
			return
		}

		// Assume rref is fqn and disect (if it's ID validation will fail later)
		parentFqn, name, err := types.DisectFqn(rref)
		if err != nil {
			render.InternalError(w, err)
			log.Debug().
				Err(err).
				Msgf("Failed to desict rref '%s'.", rref)
			return
		} else if parentFqn == "" {
			render.BadRequest(w, errors.New("A repository has to be created within a space."))
			return
		}

		// get the id of the parent space (and fail if it doesn't exist)
		parentSpace, err := spaces.FindFqn(ctx, parentFqn)
		if err != nil {
			render.BadRequest(w, err)
			log.Debug().
				Err(err).
				Msgf("Parent space '%s' doesn't exist.", parentFqn)
			return
		}

		/*
		 * AUTHORIZATION
		 * Create is a special case - check permission without specific resource
		 */
		scope := &types.Scope{SpaceFqn: parentFqn}
		resource := &types.Resource{
			Type: enum.ResourceTypeRepo,
			Name: "",
		}
		if !guard.Enforce(w, r, scope, resource, enum.PermissionRepoCreate) {
			return
		}

		in := new(repoCreateInput)
		err = json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequest(w, err)
			log.Debug().Err(err).
				Msg("Decoding json body failed.")
			return
		}

		// get current user
		usr, _ := request.UserFrom(ctx)

		// create repo
		repo := &types.Repository{
			Name:        strings.ToLower(name),
			SpaceId:     parentSpace.ID,
			Fqn:         strings.ToLower(rref),
			DisplayName: in.DisplayName,
			Description: in.Description,
			IsPublic:    in.IsPublic,
			CreatedBy:   usr.ID,
			Created:     time.Now().UnixMilli(),
			Updated:     time.Now().UnixMilli(),
			ForkId:      in.ForkId,
		}

		if ok, err := check.Repo(repo); !ok {
			render.BadRequest(w, err)
			log.Debug().Err(err).
				Msg("Repository validation failed.")
			return
		}

		// TODO: Ensure forkId exists!
		err = repos.Create(ctx, repo)
		if err != nil {
			render.InternalError(w, err)
			log.Error().Err(err).
				Msg("Repository creation failed")
		} else {
			render.JSON(w, repo, 200)
		}
	}
}
