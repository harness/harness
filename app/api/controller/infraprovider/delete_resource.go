package infraprovider

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) DeleteResource(
	ctx context.Context,
	session *auth.Session,
	spaceID int64,
	infraProviderConfigIdentifier string,
	infraProviderResourceIdentifier string,
) error {
	space, err := c.spaceFinder.FindByID(ctx, spaceID)
	if err != nil {
		return fmt.Errorf("failed to find space: %w", err)
	}
	err = apiauth.CheckInfraProvider(
		ctx,
		c.authorizer,
		session,
		space.Path,
		"",
		enum.PermissionInfraProviderDelete,
	)
	if err != nil {
		return fmt.Errorf("failed to authorize: %w", err)
	}
	return c.infraproviderSvc.DeleteResource(ctx, spaceID, infraProviderConfigIdentifier,
		infraProviderResourceIdentifier, true)
}
