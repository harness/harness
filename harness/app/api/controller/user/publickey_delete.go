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

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) DeletePublicKey(
	ctx context.Context,
	session *auth.Session,
	userUID string,
	identifier string,
) error {
	user, err := c.principalStore.FindUserByUID(ctx, userUID)
	if err != nil {
		return fmt.Errorf("failed to fetch user by uid: %w", err)
	}

	if err = apiauth.CheckUser(ctx, c.authorizer, session, user, enum.PermissionUserEdit); err != nil {
		return err
	}

	err = c.publicKeyStore.DeleteByIdentifier(ctx, user.ID, identifier)
	if err != nil {
		return fmt.Errorf("failed to delete public key by id: %w", err)
	}

	return nil
}
