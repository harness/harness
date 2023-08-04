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

	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

var (
	// errRepositoryRequiresParent if the user tries to create a repo without a parent space.
	errPipelineRequiresParent = usererror.BadRequest(
		"Parent space required - standalone pipelines are not supported.")
)

type CreateInput struct {
	Description   string       `json:"description"`
	ParentRef     string       `json:"parent_ref"` // Ref of the parent space
	UID           string       `json:"uid"`
	RepoRef       string       `json:"repo_ref"` // null if repo_type != gitness
	RepoType      enum.ScmType `json:"repo_type"`
	DefaultBranch string       `json:"default_branch"`
	ConfigPath    string       `json:"config_path"`
}

// Create creates a new pipeline
func (c *Controller) Create(ctx context.Context, session *auth.Session, in *CreateInput) (*types.Pipeline, error) {
	// TODO: Add auth
	// parentSpace, err := c.getSpaceCheckAuthRepoCreation(ctx, session, in.ParentRef)
	// if err != nil {
	// 	return nil, err
	// }

	parentSpace, err := c.spaceStore.FindByRef(ctx, in.ParentRef)
	if err != nil {
		return nil, fmt.Errorf("could not find parent by ref: %w", err)
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
	err = dbtx.New(c.db).WithTx(ctx, func(ctx context.Context) error {
		// lock parent space path to ensure it doesn't get updated while we setup new pipeline
		_, err := c.pathStore.FindPrimaryWithLock(ctx, enum.PathTargetTypeSpace, parentSpace.ID)
		if err != nil {
			return usererror.BadRequest("Parent not found")
		}

		now := time.Now().UnixMilli()
		pipeline = &types.Pipeline{
			Description:   in.Description,
			ParentID:      parentSpace.ID,
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
			return fmt.Errorf("pipeline creation failed: %w", err)
		}
		return nil
	})

	return pipeline, nil
}

func (c *Controller) sanitizeCreateInput(in *CreateInput) error {
	parentRefAsID, err := strconv.ParseInt(in.ParentRef, 10, 64)

	if (err == nil && parentRefAsID <= 0) || (len(strings.TrimSpace(in.ParentRef)) == 0) {
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
