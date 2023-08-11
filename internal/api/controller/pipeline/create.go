// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pipeline

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

var (
	// errPipelineRequiresParent if the user tries to create a pipeline without a parent space.
	errPipelineRequiresParent = usererror.BadRequest(
		"Parent space required - standalone pipelines are not supported.")
)

type CreateInput struct {
	Description   string       `json:"description"`
	SpaceRef      string       `json:"space_ref"`
	UID           string       `json:"uid"`
	RepoRef       string       `json:"repo_ref"` // empty if repo_type != gitness
	RepoType      enum.ScmType `json:"repo_type"`
	DefaultBranch string       `json:"default_branch"`
	ConfigPath    string       `json:"config_path"`
}

func (c *Controller) Create(ctx context.Context, session *auth.Session, in *CreateInput) (*types.Pipeline, error) {
	parentSpace, err := c.spaceStore.FindByRef(ctx, in.SpaceRef)
	if err != nil {
		return nil, fmt.Errorf("could not find parent by ref: %w", err)
	}

	err = apiauth.CheckPipeline(ctx, c.authorizer, session, parentSpace.Path, in.UID, enum.PermissionPipelineEdit)
	if err != nil {
		return nil, err
	}

	var repoID int64

	if in.RepoType == enum.ScmTypeGitness {
		repo, err := c.repoStore.FindByRef(ctx, in.RepoRef)
		if err != nil {
			return nil, fmt.Errorf("could not find repo by ref: %w", err)
		}
		repoID = repo.ID
	}

	if err := c.sanitizeCreateInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	var pipeline *types.Pipeline
	now := time.Now().UnixMilli()
	pipeline = &types.Pipeline{
		Description:   in.Description,
		SpaceID:       parentSpace.ID,
		UID:           in.UID,
		Seq:           0,
		RepoID:        repoID,
		RepoType:      in.RepoType,
		DefaultBranch: in.DefaultBranch,
		ConfigPath:    in.ConfigPath,
		Created:       now,
		Updated:       now,
		Version:       0,
	}
	err = c.pipelineStore.Create(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("pipeline creation failed: %w", err)
	}

	return pipeline, nil
}

func (c *Controller) sanitizeCreateInput(in *CreateInput) error {
	parentRefAsID, err := strconv.ParseInt(in.SpaceRef, 10, 64)
	if (err == nil && parentRefAsID <= 0) || (len(strings.TrimSpace(in.SpaceRef)) == 0) {
		return errPipelineRequiresParent
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
