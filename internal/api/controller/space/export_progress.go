package space

import (
	"context"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type ExportProgressOutput struct {
	Repos []types.JobProgress `json:"repos"`
}

// ExportProgress returns progress of the export job.
func (c *Controller) ExportProgress(ctx context.Context,
	session *auth.Session,
	spaceRef string,
) (*ExportProgressOutput, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceView, false); err != nil {
		return nil, err
	}

	progress, err := c.exporter.GetProgressForSpace(ctx, space.ID)
	if err != nil {
		return nil, err
	}

	if len(progress) == 0 {
		return nil, usererror.NotFound("No ongoing export found for space.")
	}

	return &ExportProgressOutput{Repos: progress}, nil
}
