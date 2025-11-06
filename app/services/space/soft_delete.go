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

package space

import (
	"context"
	"fmt"
	"math"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (s *Service) SoftDeleteInner(
	ctx context.Context,
	session *auth.Session,
	space *types.Space,
	deletedAt int64,
) error {
	filter := &types.SpaceFilter{
		Page:              1,
		Size:              math.MaxInt,
		Query:             "",
		Order:             enum.OrderAsc,
		Sort:              enum.SpaceAttrCreated,
		DeletedBeforeOrAt: nil, // only filter active subspaces
		Recursive:         true,
	}
	subSpaces, err := s.spaceStore.List(ctx, space.ID, filter)
	if err != nil {
		return fmt.Errorf("failed to list space %d sub spaces recursively: %w", space.ID, err)
	}

	allSpaces := []*types.Space{space}
	allSpaces = append(allSpaces, subSpaces...)

	if s.gitspaceSvs != nil {
		err = s.gitspaceSvs.DeleteAllForSpaces(ctx, allSpaces)
		if err != nil {
			return fmt.Errorf("failed to soft delete gitspaces of space %d: %w", space.ID, err)
		}
	}

	if s.infraProviderSvc != nil {
		err = s.infraProviderSvc.DeleteAllForSpaces(ctx, allSpaces)
		if err != nil {
			return fmt.Errorf("failed to soft delete infra providers of space %d: %w", space.ID, err)
		}
	}

	for _, space := range subSpaces {
		_, err := s.spaceStore.FindForUpdate(ctx, space.ID)
		if err != nil {
			return fmt.Errorf("failed to lock the space for update: %w", err)
		}

		if err := s.spaceStore.SoftDelete(ctx, space, deletedAt); err != nil {
			return fmt.Errorf("failed to soft delete subspace: %w", err)
		}
	}

	if s.repoStore != nil && s.repoCtrl != nil {
		err = s.softDeleteRepositoriesNoAuth(ctx, session, space.ID, deletedAt)
		if err != nil {
			return fmt.Errorf("failed to soft delete repositories of space %d: %w", space.ID, err)
		}
	}

	if err = s.spaceStore.SoftDelete(ctx, space, deletedAt); err != nil {
		return fmt.Errorf("spaceStore failed to soft delete space: %w", err)
	}

	err = s.spacePathStore.DeletePathsAndDescendandPaths(ctx, space.ID)
	if err != nil {
		return fmt.Errorf("spacePathStore failed to delete descendant paths of %d: %w", space.ID, err)
	}

	return nil
}

// softDeleteRepositoriesNoAuth soft deletes all repositories in a space - no authorization is verified.
// WARNING For internal calls only.
func (s *Service) softDeleteRepositoriesNoAuth(
	ctx context.Context,
	session *auth.Session,
	spaceID int64,
	deletedAt int64,
) error {
	filter := &types.RepoFilter{
		Page:              1,
		Size:              int(math.MaxInt),
		Query:             "",
		Order:             enum.OrderAsc,
		Sort:              enum.RepoAttrNone,
		DeletedBeforeOrAt: nil, // only filter active repos
		Recursive:         true,
	}
	repos, err := s.repoStore.List(ctx, spaceID, filter)
	if err != nil {
		return fmt.Errorf("failed to list space repositories: %w", err)
	}

	for _, repo := range repos {
		err = s.repoCtrl.SoftDeleteNoAuth(ctx, session, repo, deletedAt)
		if err != nil {
			return fmt.Errorf("failed to soft delete repository: %w", err)
		}
	}
	return nil
}
