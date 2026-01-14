// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package user

import (
	"context"
	"fmt"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/token"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

type CreateTokenInput struct {
	// TODO [CODE-1363]: remove after identifier migration.
	UID        string         `json:"uid" deprecated:"true"`
	Identifier string         `json:"identifier"`
	Lifetime   *time.Duration `json:"lifetime"`
}

/*
 * CreateToken creates a new user access token.
 */
func (c *Controller) CreateAccessToken(
	ctx context.Context,
	session *auth.Session,
	userUID string,
	in *CreateTokenInput,
) (*types.TokenResponse, error) {
	if err := c.sanitizeCreateTokenInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	user, err := findUserFromUID(ctx, c.principalStore, userUID)
	if err != nil {
		return nil, err
	}

	// Ensure principal has required permissions on parent
	if err = apiauth.CheckUser(ctx, c.authorizer, session, user, enum.PermissionUserEdit); err != nil {
		return nil, err
	}

	token, jwtToken, err := token.CreatePAT(
		ctx,
		c.tokenStore,
		&session.Principal,
		user,
		in.Identifier,
		in.Lifetime,
	)
	if err != nil {
		return nil, err
	}

	return &types.TokenResponse{Token: *token, AccessToken: jwtToken}, nil
}

func (c *Controller) sanitizeCreateTokenInput(in *CreateTokenInput) error {
	// TODO [CODE-1363]: remove after identifier migration.
	if in.Identifier == "" {
		in.Identifier = in.UID
	}

	if err := check.Identifier(in.Identifier); err != nil {
		return err
	}

	//nolint:revive
	if err := check.TokenLifetime(in.Lifetime, true); err != nil {
		return err
	}

	return nil
}
