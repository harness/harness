package space

import (
	"context"
	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ExportProgress returns progress of the export job.
func (c *Controller) ExportProgress(ctx context.Context,
	session *auth.Session,
	spaceRef string,
) (types.JobProgress, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)

	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceView, false); err != nil {
		return types.JobProgress{}, err
	}

	return c.exporter.GetProgress(ctx, space)
}
