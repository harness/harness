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

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

/*
 * DeleteToken deletes a token of a user.
 */
func (c *Controller) DeleteToken(
	ctx context.Context,
	session *auth.Session,
	userUID string,
	tokenType enum.TokenType,
	tokenIdentifier string) error {
	user, err := findUserFromUID(ctx, c.principalStore, userUID)
	if err != nil {
		return err
	}

	// Ensure principal has required permissions on parent.
	if err = apiauth.CheckUser(ctx, c.authorizer, session, user, enum.PermissionUserEdit); err != nil {
		return err
	}

	token, err := c.tokenStore.FindByIdentifier(ctx, user.ID, tokenIdentifier)
	if err != nil {
		return err
	}

	// Ensure token type matches the requested type and is a valid user token type
	if !isUserTokenType(token.Type) || token.Type != tokenType {
		// throw a not found error - no need for user to know about token.
		return usererror.ErrNotFound
	}

	// Ensure token belongs to user.
	if token.PrincipalID != user.ID {
		log.Warn().Msg("Principal tried to delete token that doesn't belong to the user")

		// throw a not found error - no need for user to know about token.
		return usererror.ErrNotFound
	}

	return c.tokenStore.Delete(ctx, token.ID)
}
