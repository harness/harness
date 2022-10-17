// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/harness/gitness/resources"

	"github.com/harness/gitness/internal/gitrpc"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
	zerolog "github.com/rs/zerolog/log"
)

type CreateInput struct {
	PathName    string `json:"pathName"`
	SpaceID     int64  `json:"spaceId"`
	Name        string `json:"name"`
	Branch      string `json:"branch"`
	Description string `json:"description"`
	IsPublic    bool   `json:"isPublic"`
	ForkID      int64  `json:"forkId"`
	Readme      bool   `json:"readme"`
	License     string `json:"license"`
	GitIgnore   string `json:"gitIgnore"`
}

// Create creates a new repository.
//
//nolint:funlen // needs refactor
func (c *Controller) Create(ctx context.Context, session *auth.Session, in *CreateInput) (*types.Repository, error) {
	log := zerolog.Ctx(ctx)
	// ensure we reference a space
	if in.SpaceID <= 0 {
		return nil, usererror.BadRequest("A repository can't exist by itself.")
	}

	parentSpace, err := c.spaceStore.Find(ctx, in.SpaceID)
	if err != nil {
		log.Err(err).Msgf("Failed to get space with id '%d'.", in.SpaceID)
		return nil, usererror.BadRequest("Parent not found'")
	}
	/*
	 * AUTHORIZATION
	 * Create is a special case - check permission without specific resource
	 */
	scope := &types.Scope{SpacePath: parentSpace.Path}
	resource := &types.Resource{
		Type: enum.ResourceTypeRepo,
		Name: "",
	}

	err = apiauth.Check(ctx, c.authorizer, session, scope, resource, enum.PermissionRepoCreate)
	if err != nil {
		return nil, fmt.Errorf("auth check failed: %w", err)
	}

	// set default branch in case it wasn't passed
	if in.Branch == "" {
		in.Branch = c.defaultBranch
	}

	// create new repo object
	repo := &types.Repository{
		PathName:      strings.ToLower(in.PathName),
		SpaceID:       in.SpaceID,
		Name:          in.Name,
		Description:   in.Description,
		IsPublic:      in.IsPublic,
		CreatedBy:     session.Principal.ID,
		Created:       time.Now().UnixMilli(),
		Updated:       time.Now().UnixMilli(),
		ForkID:        in.ForkID,
		DefaultBranch: in.Branch,
	}

	// validate repo
	if err = check.Repo(repo); err != nil {
		return nil, err
	}
	var content []byte
	files := make([]gitrpc.File, 0, 3) // readme, gitignore, licence
	if in.Readme {
		content = createReadme(in.Name, in.Description)
		files = append(files, gitrpc.File{
			Path:    "README.md",
			Content: content,
		})
	}

	if in.License != "" && in.License != "none" {
		content, err = resources.ReadLicense(in.License)
		if err != nil {
			return nil, err
		}
		files = append(files, gitrpc.File{
			Path:    "LICENSE",
			Content: content,
		})
	}

	if in.GitIgnore != "" {
		content, err = resources.ReadGitIgnore(in.GitIgnore)
		if err != nil {
			return nil, err
		}
		files = append(files, gitrpc.File{
			Path:    ".gitignore",
			Content: content,
		})
	}

	resp, err := c.gitRPCClient.CreateRepository(ctx, &gitrpc.CreateRepositoryParams{
		DefaultBranch: repo.DefaultBranch,
		Files:         files,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating repository: %w", err)
	}

	repo.GitUID = resp.UID

	// create in store
	err = c.repoStore.Create(ctx, repo)
	if err != nil {
		log.Error().Err(err).
			Msg("Repository creation failed.")

		// TODO: cleanup git repo!

		return nil, err
	}

	return repo, nil
}

func createReadme(name, description string) []byte {
	content := bytes.Buffer{}
	content.WriteString("#" + name + "\n")
	if description != "" {
		content.WriteString(description)
	}
	return content.Bytes()
}
