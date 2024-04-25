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
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (s *Service) getPublicResource(
	ctx context.Context,
	scope *types.Scope,
	resource *types.Resource,
) (*types.PublicResource, error) {
	resType := resource.Type

	// set scope type to space for checks within space scope.
	if resource.Identifier == "" {
		resType = enum.ResourceTypeSpace
	}
	var pubRes *types.PublicResource
	var err error
	switch resType {
	case enum.ResourceTypeRepo:
		pubRes, err = s.getResourceRepo(ctx, scope, resource)
	case enum.ResourceTypeSpace:
		pubRes, err = s.getResourceSpace(ctx, scope, resource)
	default:
		return nil, fmt.Errorf("invalid public resource type")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get public resource: %w", err)
	}

	return pubRes, nil
}

func (s *Service) getResourceRepo(
	ctx context.Context,
	scope *types.Scope,
	resource *types.Resource,
) (*types.PublicResource, error) {
	repoRef := paths.Concatenate(scope.SpacePath, resource.Identifier)
	repo, err := s.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo: %w", err)
	}

	return &types.PublicResource{
		Type: enum.PublicResourceTypeRepo,
		ID:   repo.ID,
	}, nil
}

func (s *Service) getResourceSpace(
	ctx context.Context,
	scope *types.Scope,
	resource *types.Resource,
) (*types.PublicResource, error) {
	spaceRef := paths.Concatenate(scope.SpacePath, resource.Identifier)
	space, err := s.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find space: %w", err)
	}

	return &types.PublicResource{
		Type: enum.PublicResourceTypeSpace,
		ID:   space.ID,
	}, nil
}
