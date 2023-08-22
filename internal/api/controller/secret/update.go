// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package secret

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// UpdateInput is used for updating a repo.
type UpdateInput struct {
	Description string `json:"description"`
	UID         string `json:"uid"`
	Data        string `json:"data"`
}

func (c *Controller) Update(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	uid string,
	in *UpdateInput,
) (*types.Secret, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find space: %w", err)
	}

	err = apiauth.CheckSecret(ctx, c.authorizer, session, space.Path, uid, enum.PermissionSecretEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize: %w", err)
	}

	secret, err := c.secretStore.FindByUID(ctx, space.ID, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to find secret: %w", err)
	}

	return c.secretStore.UpdateOptLock(ctx, secret, func(original *types.Secret) error {
		if in.Description != "" {
			original.Description = in.Description
		}
		if in.Data != "" {
			data, err := c.encrypter.Encrypt(original.Data)
			if err != nil {
				return fmt.Errorf("could not encrypt secret: %w", err)
			}
			original.Data = string(data)
		}
		if in.UID != "" {
			original.UID = in.UID
		}

		return nil
	})
}
