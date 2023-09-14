// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package serviceaccount

import (
	"context"
	"time"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/token"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

type CreateTokenInput struct {
	UID      string         `json:"uid"`
	Lifetime *time.Duration `json:"lifetime"`
}

// CreateToken creates a new service account access token.
func (c *Controller) CreateToken(
	ctx context.Context,
	session *auth.Session,
	saUID string,
	in *CreateTokenInput,
) (*types.TokenResponse, error) {
	sa, err := findServiceAccountFromUID(ctx, c.principalStore, saUID)
	if err != nil {
		return nil, err
	}

	if err = check.UID(in.UID); err != nil {
		return nil, err
	}
	if err = check.TokenLifetime(in.Lifetime, true); err != nil {
		return nil, err
	}

	// Ensure principal has required permissions on parent (ensures that parent exists)
	if err = apiauth.CheckServiceAccount(ctx, c.authorizer, session, c.spaceStore, c.repoStore,
		sa.ParentType, sa.ParentID, sa.UID, enum.PermissionServiceAccountEdit); err != nil {
		return nil, err
	}
	token, jwtToken, err := token.CreateSAT(
		ctx,
		c.tokenStore,
		&session.Principal,
		sa,
		in.UID,
		in.Lifetime,
	)
	if err != nil {
		return nil, err
	}

	return &types.TokenResponse{Token: *token, AccessToken: jwtToken}, nil
}
