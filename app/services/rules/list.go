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

package rules

import (
	"context"
	"fmt"

	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// List returns protection rules for a scope.
func (s *Service) List(ctx context.Context,
	parentID int64,
	parentType enum.RuleParent,
	inherited bool,
	filter *types.RuleFilter,
) ([]types.Rule, int64, error) {
	var parents []types.RuleParentInfo
	var err error

	switch parentType {
	case enum.RuleParentRepo:
		parents, err = s.getParentInfoRepo(ctx, parentID, inherited)
		if err != nil {
			return nil, 0, err
		}
	case enum.RuleParentSpace:
		parents, err = s.getParentInfoSpace(ctx, parentID, inherited)
		if err != nil {
			return nil, 0, err
		}
	default:
		return nil, 0, fmt.Errorf("webhook type %s is not supported", parentType)
	}

	var list []types.Rule
	var count int64

	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		list, err = s.ruleStore.List(ctx, parents, filter)
		if err != nil {
			return fmt.Errorf("failed to list protection rules: %w", err)
		}

		if filter.Page == 1 && len(list) < filter.Size {
			count = int64(len(list))
			return nil
		}

		count, err = s.ruleStore.Count(ctx, parents, filter)
		if err != nil {
			return fmt.Errorf("failed to count protection rules: %w", err)
		}

		return nil
	}, dbtx.TxDefaultReadOnly)
	if err != nil {
		return nil, 0, err
	}

	for i := range list {
		list[i].Users, list[i].UserGroups, err = s.getRuleUserAndUserGroups(ctx, &list[i])
		if err != nil {
			return nil, 0, err
		}
	}

	return list, count, nil
}

func (s *Service) getParentInfoRepo(
	ctx context.Context,
	repoID int64,
	inherited bool,
) ([]types.RuleParentInfo, error) {
	var parents []types.RuleParentInfo

	parents = append(parents, types.RuleParentInfo{
		ID:   repoID,
		Type: enum.RuleParentRepo,
	})

	if inherited {
		repo, err := s.repoStore.Find(ctx, repoID)
		if err != nil {
			return nil, fmt.Errorf("failed to get repo: %w", err)
		}

		ids, err := s.spaceStore.GetAncestorIDs(ctx, repo.ParentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent space ids: %w", err)
		}

		for _, id := range ids {
			parents = append(parents, types.RuleParentInfo{
				Type: enum.RuleParentSpace,
				ID:   id,
			})
		}
	}

	return parents, nil
}

func (s *Service) getParentInfoSpace(
	ctx context.Context,
	spaceID int64,
	inherited bool,
) ([]types.RuleParentInfo, error) {
	var parents []types.RuleParentInfo

	if inherited {
		ids, err := s.spaceStore.GetAncestorIDs(ctx, spaceID)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent space ids: %w", err)
		}

		for _, id := range ids {
			parents = append(parents, types.RuleParentInfo{
				Type: enum.RuleParentSpace,
				ID:   id,
			})
		}
	} else {
		parents = append(parents, types.RuleParentInfo{
			Type: enum.RuleParentSpace,
			ID:   spaceID,
		})
	}

	return parents, nil
}
