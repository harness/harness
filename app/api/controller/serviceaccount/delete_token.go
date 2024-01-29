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

package serviceaccount

import (
	"context"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// DeleteToken deletes a token of a service account.
func (c *Controller) DeleteToken(
	ctx context.Context,
	session *auth.Session,
	saUID string,
	identifier string,
) error {
	sa, err := findServiceAccountFromUID(ctx, c.principalStore, saUID)
	if err != nil {
		return err
	}

	// Ensure principal has required permissions on parent (ensures that parent exists)
	if err = apiauth.CheckServiceAccount(ctx, c.authorizer, session, c.spaceStore, c.repoStore,
		sa.ParentType, sa.ParentID, sa.UID, enum.PermissionServiceAccountEdit); err != nil {
		return err
	}

	token, err := c.tokenStore.FindByIdentifier(ctx, sa.ID, identifier)
	if err != nil {
		return err
	}

	// Ensure sat belongs to service account
	if token.Type != enum.TokenTypeSAT || token.PrincipalID != sa.ID {
		log.Warn().Msg("Principal tried to delete token that doesn't belong to the service account")

		// throw a not found error - no need for user to know about token?
		return usererror.ErrNotFound
	}

	return c.tokenStore.Delete(ctx, token.ID)
}
