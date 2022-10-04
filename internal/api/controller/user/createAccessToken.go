// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

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
	Name     string           `json:"name"`
	Lifetime time.Duration    `json:"lifetime"`
	Grants   enum.AccessGrant `json:"grants"`
}

/*
 * CreateToken creates a new user access token.
 */
func (c *Controller) CreateAccessToken(ctx context.Context, session *auth.Session,
	userUID string, in *CreateTokenInput) (*types.TokenResponse, error) {
	user, err := findUserFromUID(ctx, c.userStore, userUID)
	if err != nil {
		return nil, err
	}

	// Ensure principal has required permissions on parent
	if err = apiauth.CheckUser(ctx, c.authorizer, session, user, enum.PermissionUserEdit); err != nil {
		return nil, err
	}

	if err = check.Name(in.Name); err != nil {
		return nil, err
	}
	if err = check.TokenLifetime(in.Lifetime); err != nil {
		return nil, err
	}

	token, jwtToken, err := token.CreatePAT(ctx, c.tokenStore, &session.Principal,
		user, in.Name, in.Lifetime, in.Grants)
	if err != nil {
		return nil, err
	}

	return &types.TokenResponse{Token: *token, AccessToken: jwtToken}, nil
}
