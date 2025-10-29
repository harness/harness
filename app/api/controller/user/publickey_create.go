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
	Identifier string               `json:"identifier"`
	Usage      enum.PublicKeyUsage  `json:"usage"`
	Scheme     enum.PublicKeyScheme `json:"scheme"`
	Content    string               `json:"content"`
}

func (in *CreatePublicKeyInput) Sanitize() error {
	if err := check.Identifier(in.Identifier); err != nil {
		return err
	}

	if _, ok := in.Usage.Sanitize(); !ok {
		return errors.InvalidArgument("invalid value for public key usage")
	}

	if _, ok := in.Scheme.Sanitize(); !ok {
		return errors.InvalidArgument("invalid value for public key scheme")
	}

	in.Content = strings.TrimSpace(in.Content)
	if in.Content == "" {
		return errors.InvalidArgument("public key not provided")
	}

	return nil
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

	if err := in.Sanitize(); err != nil {
		return nil, err
	}

	key, err := publickey.ParseString(in.Content, &session.Principal)
	if err != nil {
		return nil, errors.InvalidArgument("unrecognized key content")
	}

	if in.Scheme != "" && key.Scheme() != in.Scheme {
		return nil, errors.InvalidArgumentf("key is not a valid %s key", in.Scheme)
	}

	switch key.Scheme() {
	case enum.PublicKeySchemeSSH:
		if in.Usage == "" {
			in.Usage = enum.PublicKeyUsageAuth // backward compatibility, default usage for SSH is auth only
		}
	case enum.PublicKeySchemePGP:
		if in.Usage == "" {
			in.Usage = enum.PublicKeyUsageSign
		} else if in.Usage != enum.PublicKeyUsageSign {
			return nil, errors.InvalidArgument(
				"invalid key usage: PGP keys can only be used for verification of signatures")
		}
	default:
		return nil, errors.InvalidArgument("unrecognized public key scheme")
	}

	now := time.Now().UnixMilli()

	k := &types.PublicKey{
		PrincipalID:      user.ID,
		Created:          now,
		Verified:         nil, // the key is created as unverified
		Identifier:       in.Identifier,
		Usage:            in.Usage,
		Fingerprint:      key.Fingerprint(),
		Content:          in.Content,
		Comment:          key.Comment(),
		Type:             key.Type(),
		Scheme:           key.Scheme(),
		ValidFrom:        key.ValidFrom(),
		ValidTo:          key.ValidTo(),
		RevocationReason: key.RevocationReason(),
		Metadata:         key.Metadata(),
	}

	keyIDs := key.KeyIDs()

	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := c.checkKeyExistence(ctx, user.ID, key, k); err != nil {
			return err
		}

		if err = c.publicKeyStore.Create(ctx, k); err != nil {
			return fmt.Errorf("failed to insert public key: %w", err)
		}

		if err = c.publicKeySubKeyStore.Create(ctx, k.ID, keyIDs); err != nil {
			return fmt.Errorf("failed to insert public key subkey: %w", err)
		}

		// If the uploaded key (or a subkey) is revoked (only possible with PGP keys) with reason=compromised
		// then we revoke all existing signatures in the DB signed with the key.
		compromisedKeyIDs := key.CompromisedIDs()
		if len(compromisedKeyIDs) > 0 {
			if err = c.gitSignatureResultStore.UpdateAll(
				ctx,
				enum.GitSignatureRevoked,
				user.ID,
				compromisedKeyIDs, nil,
			); err != nil {
				return fmt.Errorf("failed to revoke all PGP signatures signed with compromised keys %v: %w",
					compromisedKeyIDs, err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return k, nil
}

func (c *Controller) checkKeyExistence(
	ctx context.Context,
	userID int64,
	key publickey.KeyInfo,
	k *types.PublicKey,
) error {
	schemes := []enum.PublicKeyScheme{key.Scheme()}
	switch key.Scheme() {
	case enum.PublicKeySchemeSSH:
		// For SSH keys we don't allow the same key twice, even for two different users.
		// The fingerprint field is indexed in the DB, but it's not a unique index.
		existingKeys, err := c.publicKeyStore.ListByFingerprint(ctx, k.Fingerprint, nil, nil, schemes)
		if err != nil {
			return fmt.Errorf("failed to read keys by fingerprint: %w", err)
		}

		for _, existingKey := range existingKeys {
			if key.Matches(existingKey.Content) {
				return errors.InvalidArgument("key is already in use")
			}
		}
	case enum.PublicKeySchemePGP:
		// For PGP keys we don't allow the same key twice for the same user.
		existingKeys, err := c.publicKeyStore.ListByFingerprint(ctx, k.Fingerprint, &userID, nil, schemes)
		if err != nil {
			return fmt.Errorf("failed to read keys by userID and fingerprint: %w", err)
		}

		if len(existingKeys) > 0 {
			return errors.InvalidArgument("key is already in use")
		}
	}

	return nil
}
