// Copyright 2021 Harness Inc. All rights reserved.
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
//nolint:funlen,goimports // needs refactor
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

	// create new repo object
	repo := &types.Repository{
		PathName:    strings.ToLower(in.PathName),
		SpaceID:     in.SpaceID,
		Name:        in.Name,
		Description: in.Description,
		IsPublic:    in.IsPublic,
		CreatedBy:   session.Principal.ID,
		Created:     time.Now().UnixMilli(),
		Updated:     time.Now().UnixMilli(),
		ForkID:      in.ForkID,
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
			Name:    "README.md",
			Base64:  false,
			Content: content,
		})
	}

	if in.License != "" && in.License != "none" {
		// TODO: The caller shouldn't need to know the actual location.
		content, err = resources.Licence.ReadFile(fmt.Sprintf("license/%s.txt", in.License))
		if err != nil {
			return nil, err
		}
		files = append(files, gitrpc.File{
			Name:    "LICENSE",
			Base64:  false,
			Content: content,
		})
	}

	if in.GitIgnore != "" {
		// TODO: The caller shouldn't need to know the actual location.
		content, err = resources.Gitignore.ReadFile(fmt.Sprintf("gitignore/%s.gitignore", in.GitIgnore))
		if err != nil {
			return nil, err
		}
		files = append(files, gitrpc.File{
			Name:    ".gitignore",
			Base64:  false,
			Content: content,
		})
	}

	err = c.rpcClient.CreateRepository(ctx, &gitrpc.CreateRepositoryParams{
		RepositoryParams: gitrpc.RepositoryParams{
			Username: session.Principal.Name,
			// TODO: use UID as name
			Name:   repo.PathName,
			Branch: in.Branch,
		},
		Files: files,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating repository: %w", err)
	}

	// create in store
	err = c.repoStore.Create(ctx, repo)
	if err != nil {
		log.Error().Err(err).
			Msg("Repository creation failed.")

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
