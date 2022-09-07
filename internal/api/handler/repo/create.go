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
	Name        string `json:"name"`
	SpaceId     int64  `json:"spaceId"`
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

		in := new(repoCreateInput)
		err := json.NewDecoder(r.Body).Decode(in)
		if err != nil {
			render.BadRequest(w, err)
			log.Debug().Err(err).
				Msg("Decoding json body failed.")
			return
		}

		// ensure we reference a space
		if in.SpaceId <= 0 {
			render.BadRequest(w, errors.New("A repository can only be created within a space."))
			log.Debug().
				Msg("No space was provided.")
			return
		}

		parentSpace, err := spaces.Find(ctx, in.SpaceId)
		if err != nil {
			render.BadRequest(w, err)
			log.Debug().
				Err(err).
				Msgf("Parent space with id '%s' doesn't exist.", in.SpaceId)

			return
		}

		// parentFqn is assumed to be valid, in.Name gets validated in check.Repo function
		parentFqn := parentSpace.Fqn
		fqn := parentFqn + "/" + in.Name

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

		// get current user (safe to be there, or enforce would fail)
		usr, _ := request.UserFrom(ctx)

		// create repo
		repo := &types.Repository{
			Name:        strings.ToLower(in.Name),
			SpaceId:     in.SpaceId,
			Fqn:         strings.ToLower(fqn),
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
