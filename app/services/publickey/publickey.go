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

type Service interface {
	ValidateKey(ctx context.Context, publicKey ssh.PublicKey, usage enum.PublicKeyUsage) (*types.PrincipalInfo, error)
}

func NewService(
	publicKeyStore store.PublicKeyStore,
	pCache store.PrincipalInfoCache,
) LocalService {
	return LocalService{
		publicKeyStore: publicKeyStore,
		pCache:         pCache,
	}
}

type LocalService struct {
	publicKeyStore store.PublicKeyStore
	pCache         store.PrincipalInfoCache
}

// ValidateKey tries to match the provided key to one of the keys in the database.
// It updates the verified timestamp of the matched key to mark it as used.
func (s LocalService) ValidateKey(
	ctx context.Context,
	publicKey ssh.PublicKey,
	usage enum.PublicKeyUsage,
) (*types.PrincipalInfo, error) {
	key := From(publicKey)
	fingerprint := key.Fingerprint()

	existingKeys, err := s.publicKeyStore.ListByFingerprint(ctx, fingerprint)
	if err != nil {
		return nil, fmt.Errorf("failed to read keys by fingerprint: %w", err)
	}

	var keyID int64
	var principalID int64

	for _, existingKey := range existingKeys {
		if !key.Matches(existingKey.Content) || existingKey.Usage != usage {
			continue
		}

		keyID = existingKey.ID
		principalID = existingKey.PrincipalID
	}

	if keyID == 0 {
		return nil, errors.NotFound("Unrecognized key")
	}

	pInfo, err := s.pCache.Get(ctx, principalID)
	if err != nil {
		return nil, fmt.Errorf("failed to pull principal info by public key's principal ID: %w", err)
	}

	err = s.publicKeyStore.MarkAsVerified(ctx, keyID, time.Now().UnixMilli())
	if err != nil {
		return nil, fmt.Errorf("failed mark key as verified: %w", err)
	}

	return pInfo, nil
}
