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
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type UpdatePublicKeyInput struct {
	RevocationReason *enum.RevocationReason `json:"revocation_reason"`
	ValidFrom        *int64                 `json:"valid_from"`
	ValidTo          *int64                 `json:"valid_to"`
}

func (in *UpdatePublicKeyInput) Sanitize() error {
	if in.RevocationReason != nil {
		if _, ok := in.RevocationReason.Sanitize(); !ok {
			return errors.InvalidArgument("invalid public key revocation reason")
		}

		if in.ValidFrom != nil || in.ValidTo != nil {
			return errors.InvalidArgument("must either revoke the key or update its validity period")
		}
	}

	if in.ValidFrom != nil && in.ValidTo != nil {
		if *in.ValidFrom > *in.ValidTo {
			return errors.InvalidArgument("invalid validity period")
		}
	}

	return nil
}

func (c *Controller) UpdatePublicKey(
	ctx context.Context,
	session *auth.Session,
	userUID string,
	identifier string,
	in *UpdatePublicKeyInput,
) (*types.PublicKey, error) {
	user, err := c.principalStore.FindUserByUID(ctx, userUID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user by uid: %w", err)
	}

	if err = apiauth.CheckUser(ctx, c.authorizer, session, user, enum.PermissionUserEdit); err != nil {
		return nil, fmt.Errorf("access check failed: %w", err)
	}

	if err := in.Sanitize(); err != nil {
		return nil, err
	}

	key, err := c.publicKeyStore.FindByIdentifier(ctx, user.ID, identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to find public key by identifier: %w", err)
	}

	if key.RevocationReason != nil {
		if in.ValidFrom != nil || in.ValidTo != nil {
			return nil, errors.InvalidArgument("can't update the validity period of revoked keys")
		}

		if *key.RevocationReason == enum.RevocationReasonCompromised {
			return nil, errors.InvalidArgument("can't update the revocation reason of compromised keys")
		}
	}

	var (
		changedRevocationReason bool
		changedValidityPeriod   bool
	)

	if in.RevocationReason != nil &&
		(key.RevocationReason == nil || *key.RevocationReason != *in.RevocationReason) {
		now := time.Now().UnixMilli()
		key.RevocationReason = in.RevocationReason
		if key.ValidTo == nil || *key.ValidTo > now {
			key.ValidTo = &now
		}
		changedRevocationReason = true
	}

	isTimestampChanged := func(ts1, ts2 *int64) bool {
		return ts1 != nil && ts2 != nil && *ts1 != *ts2 ||
			ts1 == nil && ts2 != nil ||
			ts1 != nil && ts2 == nil
	}

	if in.ValidFrom != nil {
		if *in.ValidFrom == 0 { // zero means clear
			in.ValidFrom = nil
		}

		if isTimestampChanged(key.ValidFrom, in.ValidFrom) {
			key.ValidFrom = in.ValidFrom
			changedValidityPeriod = true
		}
	}

	if in.ValidTo != nil {
		if *in.ValidTo == 0 { // zero means clear
			in.ValidTo = nil
		}

		if isTimestampChanged(key.ValidTo, in.ValidTo) {
			key.ValidTo = in.ValidTo
			changedValidityPeriod = true
		}
	}

	if !changedRevocationReason && !changedValidityPeriod {
		return key, nil
	}

	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		err = c.publicKeyStore.Update(ctx, key)
		if err != nil {
			return fmt.Errorf("failed to update public key: %w", err)
		}

		if changedRevocationReason && *key.RevocationReason == enum.RevocationReasonCompromised {
			switch key.Scheme {
			case enum.PublicKeySchemePGP:
				subKeys, err := c.publicKeySubKeyStore.List(ctx, key.ID)
				if err != nil {
					return fmt.Errorf("failed to list subkeys: %w", err)
				}

				err = c.gitSignatureResultStore.UpdateAll(
					ctx,
					enum.GitSignatureRevoked,
					key.PrincipalID,
					subKeys,
					nil)
				if err != nil {
					return fmt.Errorf("failed to revoke all PGP signatures for keys %v: %w", subKeys, err)
				}

			case enum.PublicKeySchemeSSH:
				fingerprints := []string{key.Fingerprint}
				err := c.gitSignatureResultStore.UpdateAll(
					ctx,
					enum.GitSignatureRevoked,
					key.PrincipalID,
					nil,
					fingerprints)
				if err != nil {
					return fmt.Errorf("failed to revoke all SSH signatures for key %v: %w", fingerprints, err)
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return key, nil
}
