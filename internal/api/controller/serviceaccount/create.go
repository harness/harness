// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package serviceaccount

import (
	"context"
	"time"

	"github.com/dchest/uniuri"
	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

type CreateInput struct {
	UID        string                  `json:"uid"`
	Name       string                  `json:"name"`
	ParentType enum.ParentResourceType `json:"parentType"`
	ParentID   int64                   `json:"parentId"`
}

/*
 * Create creates a new service account.
 */
func (c *Controller) Create(ctx context.Context, session *auth.Session,
	in *CreateInput) (*types.ServiceAccount, error) {
	sa := &types.ServiceAccount{
		UID:        in.UID,
		Name:       in.Name,
		Salt:       uniuri.NewLen(uniuri.UUIDLen),
		Created:    time.Now().UnixMilli(),
		Updated:    time.Now().UnixMilli(),
		ParentType: in.ParentType,
		ParentID:   in.ParentID,
	}

	// validate service account
	if err := check.ServiceAccount(sa); err != nil {
		return nil, err
	}

	// Ensure principal has required permissions on parent (ensures that parent exists)
	// since it's a create, we use don't pass a resource name.
	if err := apiauth.CheckServiceAccount(ctx, c.authorizer, session, c.spaceStore, c.repoStore,
		sa.ParentType, sa.ParentID, "", enum.PermissionServiceAccountCreate); err != nil {
		return nil, err
	}

	// TODO: Racing condition with parent (space/repo) being deleted!
	err := c.saStore.Create(ctx, sa)
	if err != nil {
		return nil, err
	}

	return sa, nil
}
