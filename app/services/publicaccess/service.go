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
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/log"
)

var errPublicResourceCreationDisabled = errors.New("public resource creation is disabled")

type service struct {
	publicResourceCreationEnabled bool
	publicAccessStore             store.PublicAccessStore
	repoStore                     store.RepoStore
	spaceStore                    store.SpaceStore
}

func NewService(
	config *types.Config,
	publicAccessStore store.PublicAccessStore,
	repoStore store.RepoStore,
	spaceStore store.SpaceStore,
) Service {
	return &service{
		publicResourceCreationEnabled: config.PublicResourceCreationEnabled,
		publicAccessStore:             publicAccessStore,
		repoStore:                     repoStore,
		spaceStore:                    spaceStore,
	}
}

func (s *service) Get(
	ctx context.Context,
	resourceType enum.PublicResourceType,
	resourcePath string,
) (bool, error) {
	pubResID, err := s.getPublicResource(ctx, resourceType, resourcePath)
	if err != nil {
		return false, err
	}

	err = s.publicAccessStore.Find(ctx, resourceType, pubResID)
	if errors.Is(err, gitness_store.ErrResourceNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to get public access resource: %w", err)
	}

	return true, nil
}

func (s *service) Set(
	ctx context.Context,
	resourceType enum.PublicResourceType,
	resourcePath string,
	enable bool,
) error {
	if enable && !s.publicResourceCreationEnabled {
		return errPublicResourceCreationDisabled
	}

	pubResID, err := s.getPublicResource(ctx, resourceType, resourcePath)
	if err != nil {
		return err
	}

	if enable {
		err := s.publicAccessStore.Create(ctx, resourceType, pubResID)
		if errors.Is(err, gitness_store.ErrDuplicate) {
			log.Ctx(ctx).Warn().Msgf("repo %d is already set for public access", pubResID)
			return nil
		}
		return err
	} else {
		return s.publicAccessStore.Delete(ctx, resourceType, pubResID)
	}
}
