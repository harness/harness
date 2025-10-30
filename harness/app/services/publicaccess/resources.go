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
	"fmt"

	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/types/enum"
)

func (s *service) getResourceID(
	ctx context.Context,
	resourceType enum.PublicResourceType,
	resourcePath string,
) (int64, error) {
	var id int64
	var err error
	switch resourceType {
	case enum.PublicResourceTypeRepo:
		id, err = s.getResourceRepo(ctx, resourcePath)
	case enum.PublicResourceTypeSpace:
		id, err = s.getResourceSpace(ctx, resourcePath)
	case enum.PublicResourceTypeRegistry:
		id, err = s.getResourceRegistry(ctx, resourcePath)
	default:
		return 0, fmt.Errorf("invalid public resource type")
	}

	if err != nil {
		return 0, fmt.Errorf("failed to get public resource id: %w", err)
	}

	return id, nil
}

func (s *service) getResourceRepo(
	ctx context.Context,
	path string,
) (int64, error) {
	repo, err := s.repoFinder.FindByRef(ctx, path)
	if err != nil {
		return 0, fmt.Errorf("failed to find repo: %w", err)
	}

	return repo.ID, nil
}

func (s *service) getResourceSpace(
	ctx context.Context,
	path string,
) (int64, error) {
	space, err := s.spaceFinder.FindByRef(ctx, path)
	if err != nil {
		return 0, fmt.Errorf("failed to find space: %w", err)
	}

	return space.ID, nil
}

func (s *service) getResourceRegistry(
	ctx context.Context,
	path string,
) (int64, error) {
	rootRef, _, err := paths.DisectRoot(path)
	if err != nil {
		return 0, fmt.Errorf("failed to disect root from path: %w", err)
	}
	_, registryIdentifier, err := paths.DisectLeaf(path)
	if err != nil {
		return 0, fmt.Errorf("failed to disect leaf from path: %w", err)
	}
	repo, err := s.registryFinder.FindByRootRef(ctx, rootRef, registryIdentifier)
	if err != nil {
		return 0, fmt.Errorf("failed to find repo: %w", err)
	}

	return repo.ID, nil
}
