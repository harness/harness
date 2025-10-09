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

package remoteauth

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/token"
	"github.com/harness/gitness/types/enum"
)

type Service interface {
	// GenerateToken generates a jwt for the given principle to access the resource (for git-lfs-authenticate response)
	GenerateToken(
		ctx context.Context,
		principalID int64,
		principalType enum.PrincipalType,
		resource string,
	) (string, error)
}

func NewService(tokenStore store.TokenStore, principalStore store.PrincipalStore) LocalService {
	return LocalService{
		tokenStore:     tokenStore,
		principalStore: principalStore,
	}
}

type LocalService struct {
	tokenStore     store.TokenStore
	principalStore store.PrincipalStore
}

func (s LocalService) GenerateToken(
	ctx context.Context,
	principalID int64,
	_ enum.PrincipalType,
	_ string,
) (string, error) {
	identifier := token.GenerateIdentifier("remoteAuth")

	principal, err := s.principalStore.Find(ctx, principalID)
	if err != nil {
		return "", fmt.Errorf("failed to find principal %d: %w", principalID, err)
	}

	_, jwt, err := token.CreateRemoteAuthToken(ctx, s.tokenStore, principal, identifier)
	if err != nil {
		return "", fmt.Errorf("failed to create a remote auth token: %w", err)
	}

	return jwt, nil
}
