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

package publicaccess

import (
	"context"
	"errors"
	"fmt"

	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
)

type Service struct {
	publicResourceStore store.PublicResource
	repoStore           store.RepoStore
	spaceStore          store.SpaceStore
}

func NewService(
	publicResourceStore store.PublicResource,
	repoStore store.RepoStore,
	spaceStore store.SpaceStore,
) PublicAccess {
	return &Service{
		publicResourceStore: publicResourceStore,
		repoStore:           repoStore,
		spaceStore:          spaceStore,
	}
}

func (s *Service) Get(
	ctx context.Context,
	scope *types.Scope,
	resource *types.Resource,
) (bool, error) {
	pubRes, err := s.getPublicResource(ctx, scope, resource)
	if err != nil {
		return false, err
	}

	err = s.publicResourceStore.Find(ctx, pubRes)
	if errors.Is(err, gitness_store.ErrResourceNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get public access resource: %w", err)
	}

	return true, nil
}

func (s *Service) Set(
	ctx context.Context,
	scope *types.Scope,
	resource *types.Resource,
	enable bool,
) error {
	pubRes, err := s.getPublicResource(ctx, scope, resource)
	if err != nil {
		return err
	}

	if enable {
		err := s.publicResourceStore.Create(ctx, pubRes)
		if errors.Is(err, gitness_store.ErrDuplicate) {
			return nil
		}
		return err
	} else {
		return s.publicResourceStore.Delete(ctx, pubRes)
	}
}
