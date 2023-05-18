// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"context"
	"fmt"
	"math"

	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// deleteRepositoriesNoAuth does not check PermissionRepoView, and PermissionRepoDelete permissions
// Call this through Delete(Space) api to make sure the caller has DeleteSpace permission.
func (c *Controller) deleteRepositoriesNoAuth(ctx context.Context, session *auth.Session, spaceID int64) error {
	filter := &types.RepoFilter{
		Page:  1,
		Size:  int(math.MaxInt),
		Query: "",
		Order: enum.OrderAsc,
		Sort:  enum.RepoAttrNone,
	}
	repos, _, err := c.ListRepositoriesNoAuth(ctx, spaceID, filter)
	if err != nil {
		return fmt.Errorf("failed to list space repositories: %w", err)
	}
	for _, repo := range repos {
		err = c.repoCtrl.DeleteNoAuth(ctx, session, repo)
		if err != nil {
			return fmt.Errorf("failed to delete repository %d: %w", repo.ID, err)
		}
	}
	return nil
}
