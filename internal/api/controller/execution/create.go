// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package execution

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
)

var (
	// errRepositoryRequiresParent if the user tries to create a repo without a parent space.
	errPipelineRequiresParent = usererror.BadRequest(
		"Parent space required - standalone pipelines are not supported.")
)

// TODO: Add more as needed
type CreateInput struct {
	Status string `json:"status"`
}

// Create creates a new execution
func (c *Controller) Create(ctx context.Context, session *auth.Session, spaceRef string, uid string, in *CreateInput) (*types.Execution, error) {
	// TODO: Add auth
	// parentSpace, err := c.getSpaceCheckAuthRepoCreation(ctx, session, in.ParentRef)
	// if err != nil {
	// 	return nil, err
	// }

	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, fmt.Errorf("could not find space: %w", err)
	}

	if err := c.sanitizeCreateInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	var execution *types.Execution
	err = dbtx.New(c.db).WithTx(ctx, func(ctx context.Context) error {
		now := time.Now().UnixMilli()
		pipeline, err := c.pipelineStore.FindByUID(ctx, space.ID, uid)
		if err != nil {
			return err
		}
		fmt.Println("seq before: ", pipeline.Seq)
		pipeline, err = c.pipelineStore.Increment(ctx, pipeline)
		if err != nil {
			return err
		}
		fmt.Println("seq after: ", pipeline.Seq)
		execution = &types.Execution{
			Number:     pipeline.Seq,
			Status:     in.Status,
			RepoID:     pipeline.RepoID,
			PipelineID: pipeline.ID,
			Created:    now,
			Updated:    now,
			Version:    0,
		}
		err = c.executionStore.Create(ctx, execution)
		if err != nil {
			return fmt.Errorf("execution creation failed: %w", err)
		}
		return nil
	})

	return execution, nil
}

func (c *Controller) sanitizeCreateInput(in *CreateInput) error {
	return nil
}
