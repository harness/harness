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

package keyfetcher

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type Service interface {
	FetchByFingerprint(
		ctx context.Context,
		keyFingerprint string,
		principalID int64,
		usages []enum.PublicKeyUsage,
		schemes []enum.PublicKeyScheme,
	) ([]types.PublicKey, error)

	FetchBySubKeyID(
		ctx context.Context,
		subKeyID string,
		principalID int64,
		usages []enum.PublicKeyUsage,
		schemes []enum.PublicKeyScheme,
	) ([]types.PublicKey, error)
}

func NewService(
	publicKeyStore store.PublicKeyStore,
) Service {
	return service{
		publicKeyStore: publicKeyStore,
	}
}

type service struct {
	publicKeyStore store.PublicKeyStore
}

func (s service) FetchByFingerprint(
	ctx context.Context,
	keyFingerprint string,
	principalID int64,
	usages []enum.PublicKeyUsage,
	schemes []enum.PublicKeyScheme,
) ([]types.PublicKey, error) {
	keys, err := s.publicKeyStore.ListByFingerprint(ctx, keyFingerprint, &principalID, usages, schemes)
	if err != nil {
		return nil, fmt.Errorf("failed to list public keys by fingerprint: %w", err)
	}

	return keys, nil
}

func (s service) FetchBySubKeyID(
	ctx context.Context,
	subKeyID string,
	principalID int64,
	usages []enum.PublicKeyUsage,
	schemes []enum.PublicKeyScheme,
) ([]types.PublicKey, error) {
	keys, err := s.publicKeyStore.ListBySubKeyID(ctx, subKeyID, &principalID, usages, schemes)
	if err != nil {
		return nil, fmt.Errorf("failed to list public keys by subkey ID: %w", err)
	}

	return keys, nil
}
