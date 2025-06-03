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

package publickey

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gliderlabs/ssh"
)

type SSHAuthService interface {
	ValidateKey(ctx context.Context,
		username string,
		publicKey ssh.PublicKey,
	) (*types.PrincipalInfo, error)
}

func NewSSHAuthService(
	publicKeyStore store.PublicKeyStore,
	pCache store.PrincipalInfoCache,
) SSHAuthService {
	return sshAuthService{
		publicKeyStore: publicKeyStore,
		pCache:         pCache,
	}
}

type sshAuthService struct {
	publicKeyStore store.PublicKeyStore
	pCache         store.PrincipalInfoCache
}

// ValidateKey tries to match the provided SSH key to one of the keys in the database.
// It updates the verified timestamp of the matched key to mark it as used.
func (s sshAuthService) ValidateKey(
	ctx context.Context,
	_ string,
	publicKey ssh.PublicKey,
) (*types.PrincipalInfo, error) {
	key := FromSSH(publicKey)
	fingerprint := key.Fingerprint()

	existingKeys, err := s.publicKeyStore.ListByFingerprint(
		ctx,
		fingerprint,
		nil,
		[]enum.PublicKeyUsage{enum.PublicKeyUsageAuth},
		[]enum.PublicKeyScheme{enum.PublicKeySchemeSSH},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to read keys by fingerprint: %w", err)
	}

	var selectedKey *types.PublicKey
	for _, existingKey := range existingKeys {
		if key.Matches(existingKey.Content) {
			selectedKey = &existingKey
			break
		}
	}

	if selectedKey == nil {
		return nil, errors.NotFound("Unrecognized key")
	}

	if rev := selectedKey.RevocationReason; rev != nil && *rev == enum.RevocationReasonCompromised {
		return nil, errors.Forbidden("Key has been revoked")
	}

	now := time.Now().UnixMilli()

	if t := selectedKey.ValidFrom; t != nil && now < *t {
		return nil, errors.Forbidden("Key not valid")
	}
	if t := selectedKey.ValidTo; t != nil && now > *t {
		if selectedKey.RevocationReason != nil {
			return nil, errors.Forbidden("Key has been revoked")
		}
		return nil, errors.Forbidden("Key has expired")
	}

	pInfo, err := s.pCache.Get(ctx, selectedKey.PrincipalID)
	if err != nil {
		return nil, fmt.Errorf("failed to pull principal info by public key's principal ID: %w", err)
	}

	err = s.publicKeyStore.MarkAsVerified(ctx, selectedKey.ID, now)
	if err != nil {
		return nil, fmt.Errorf("failed mark key as verified: %w", err)
	}

	return pInfo, nil
}
