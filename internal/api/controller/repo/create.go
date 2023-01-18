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

	"github.com/harness/gitness/gitrpc"
	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/paths"
	"github.com/harness/gitness/internal/store/database/dbtx"
	"github.com/harness/gitness/resources"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

var (
	// errRepositoryRequiresParent if the user tries to create a repo without a parent space.
	errRepositoryRequiresParent = usererror.BadRequest(
		"Parent space required - standalone repositories are not supported.")
)

type CreateInput struct {
	ParentID      int64  `json:"parent_id"`
	UID           string `json:"uid"`
	DefaultBranch string `json:"default_branch"`
	Description   string `json:"description"`
	IsPublic      bool   `json:"is_public"`
	ForkID        int64  `json:"fork_id"`
	Readme        bool   `json:"readme"`
	License       string `json:"license"`
	GitIgnore     string `json:"git_ignore"`
}

// Create creates a new repository.
func (c *Controller) Create(ctx context.Context, session *auth.Session, in *CreateInput) (*types.Repository, error) {
	if err := c.checkAuthRepoCreation(ctx, session, in.ParentID); err != nil {
		return nil, err
	}

	if err := c.sanitizeCreateInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	gitRPCResp, err := c.createGitRPCRepository(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("error creating repository on GitRPC: %w", err)
	}

	var repo *types.Repository
	err = dbtx.New(c.db).WithTx(ctx, func(ctx context.Context) error {
		// lock parent space path to ensure it doesn't get updated while we setup new repo
		var spacePath *types.Path
		spacePath, err = c.pathStore.FindPrimaryWithLock(ctx, enum.PathTargetTypeSpace, in.ParentID)
		if err != nil {
			return usererror.BadRequest("Parent not found'")
		}

		now := time.Now().UnixMilli()
		repo = &types.Repository{
			Version:       0,
			ParentID:      in.ParentID,
			UID:           in.UID,
			GitUID:        gitRPCResp.UID,
			Path:          paths.Concatinate(spacePath.Value, in.UID),
			Description:   in.Description,
			IsPublic:      in.IsPublic,
			CreatedBy:     session.Principal.ID,
			Created:       now,
			Updated:       now,
			ForkID:        in.ForkID,
			DefaultBranch: in.DefaultBranch,
		}
		err = c.repoStore.Create(ctx, repo)
		if err != nil {
			// TODO: cleanup git repo!
			return fmt.Errorf("faild to create repository in storage: %w", err)
		}

		path := &types.Path{
			Version:    0,
			Value:      repo.Path,
			IsPrimary:  true,
			TargetType: enum.PathTargetTypeRepo,
			TargetID:   repo.ID,
			CreatedBy:  repo.CreatedBy,
			Created:    repo.Created,
			Updated:    repo.Updated,
		}
		err = c.pathStore.Create(ctx, path)
		if err != nil {
			return fmt.Errorf("failed to create path: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// backfil GitURL
	repo.GitURL = c.urlProvider.GenerateRepoCloneURL(repo.Path)

	return repo, nil
}

func (c *Controller) checkAuthRepoCreation(ctx context.Context, session *auth.Session, parentID int64) error {
	space, err := c.spaceStore.Find(ctx, parentID)
	if err != nil {
		return fmt.Errorf("parent space not found: %w", err)
	}

	// create is a special case - check permission without specific resource
	scope := &types.Scope{SpacePath: space.Path}
	resource := &types.Resource{
		Type: enum.ResourceTypeRepo,
		Name: "",
	}

	err = apiauth.Check(ctx, c.authorizer, session, scope, resource, enum.PermissionRepoCreate)
	if err != nil {
		return fmt.Errorf("auth check failed: %w", err)
	}

	return nil
}

func (c *Controller) sanitizeCreateInput(in *CreateInput) error {
	if in.ParentID <= 0 {
		return errRepositoryRequiresParent
	}

	if err := c.uidCheck(in.UID, false); err != nil {
		return err
	}

	in.Description = strings.TrimSpace(in.Description)
	if err := check.Description(in.Description); err != nil {
		return err
	}

	if in.DefaultBranch == "" {
		in.DefaultBranch = c.defaultBranch
	}

	return nil
}

func (c *Controller) createGitRPCRepository(ctx context.Context,
	in *CreateInput) (*gitrpc.CreateRepositoryOutput, error) {
	var (
		err     error
		content []byte
	)
	files := make([]gitrpc.File, 0, 3) // readme, gitignore, licence
	if in.Readme {
		content = createReadme(in.UID, in.Description)
		files = append(files, gitrpc.File{
			Path:    "README.md",
			Content: content,
		})
	}
	if in.License != "" && in.License != "none" {
		content, err = resources.ReadLicense(in.License)
		if err != nil {
			return nil, fmt.Errorf("failed to read license '%s': %w", in.License, err)
		}
		files = append(files, gitrpc.File{
			Path:    "LICENSE",
			Content: content,
		})
	}
	if in.GitIgnore != "" {
		content, err = resources.ReadGitIgnore(in.GitIgnore)
		if err != nil {
			return nil, fmt.Errorf("failed to read git ignore '%s': %w", in.GitIgnore, err)
		}
		files = append(files, gitrpc.File{
			Path:    ".gitignore",
			Content: content,
		})
	}

	resp, err := c.gitRPCClient.CreateRepository(ctx, &gitrpc.CreateRepositoryParams{
		DefaultBranch: in.DefaultBranch,
		Files:         files,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create repo on gitrpc: %w", err)
	}

	return resp, nil
}

func createReadme(name, description string) []byte {
	content := bytes.Buffer{}
	content.WriteString("# " + name + "\n")
	if description != "" {
		content.WriteString(description)
	}
	return content.Bytes()
}
