// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package logs

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/livelog"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) Tail(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	pipelineUID string,
	executionNum int64,
	stageNum int64,
	stepNum int64,
) (<-chan *livelog.Line, <-chan error, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, nil, fmt.Errorf("could not find parent space: %w", err)
	}

	pipeline, err := c.pipelineStore.FindByUID(ctx, space.ID, pipelineUID)
	if err != nil {
		return nil, nil, fmt.Errorf("could not find pipeline: %w", err)
	}

	err = apiauth.CheckPipeline(ctx, c.authorizer, session, space.Path, pipeline.UID, enum.PermissionPipelineView)
	if err != nil {
		return nil, nil, fmt.Errorf("could not authorize: %w", err)
	}

	// TODO: Figure out step ID by querying stages and steps tables
	var id int64

	linec, errc := c.logStream.Tail(ctx, id)
	return linec, errc, nil
}
