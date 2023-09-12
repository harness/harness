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
	repoRef string,
) (types.JobProgress, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return types.JobProgress{}, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoView, false); err != nil {
		return types.JobProgress{}, err
	}

	return c.exporter.GetProgress(ctx, repo)
}
