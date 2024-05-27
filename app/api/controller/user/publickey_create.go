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
	"strings"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/publickey"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

type CreatePublicKeyInput struct {
	Identifier string              `json:"identifier"`
	Usage      enum.PublicKeyUsage `json:"usage"`
	Content    string              `json:"content"`
}

func (c *Controller) CreatePublicKey(
	ctx context.Context,
	session *auth.Session,
	userUID string,
	in *CreatePublicKeyInput,
) (*types.PublicKey, error) {
	user, err := c.principalStore.FindUserByUID(ctx, userUID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user by uid: %w", err)
	}

	if err = apiauth.CheckUser(ctx, c.authorizer, session, user, enum.PermissionUserEdit); err != nil {
		return nil, err
	}

	if err := sanitizeCreatePublicKeyInput(in); err != nil {
		return nil, err
	}

	key, comment, err := publickey.ParseString(in.Content)
	if err != nil {
		return nil, errors.InvalidArgument("could not parse public key")
	}

	now := time.Now().UnixMilli()

	k := &types.PublicKey{
		PrincipalID: user.ID,
		Created:     now,
		Verified:    nil, // the key is created as unverified
		Identifier:  in.Identifier,
		Usage:       in.Usage,
		Fingerprint: key.Fingerprint(),
		Content:     in.Content,
		Comment:     comment,
		Type:        key.Type(),
	}

	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		existingKeys, err := c.publicKeyStore.ListByFingerprint(ctx, k.Fingerprint)
		if err != nil {
			return fmt.Errorf("failed to read keys by fingerprint: %w", err)
		}

		for _, existingKey := range existingKeys {
			if key.Matches(existingKey.Content) {
				return errors.InvalidArgument("Key is already in use")
			}
		}

		err = c.publicKeyStore.Create(ctx, k)
		if err != nil {
			return fmt.Errorf("failed to insert public key: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return k, nil
}

func sanitizeCreatePublicKeyInput(in *CreatePublicKeyInput) error {
	if err := check.Identifier(in.Identifier); err != nil {
		return err
	}

	usage, ok := in.Usage.Sanitize()
	if !ok {
		return errors.InvalidArgument("invalid value for public key usage")
	}
	in.Usage = usage

	in.Content = strings.TrimSpace(in.Content)
	if in.Content == "" {
		return errors.InvalidArgument("public key not provided")
	}

	return nil
}
