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
	"encoding/json"
	"errors"
	"fmt"

	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// ruleTypeToResourceType maps a protection rule type to the correct audit.ResourceType.
func ruleTypeToResourceType(ruleType enum.RuleType) audit.ResourceType {
	switch ruleType {
	case protection.TypeBranch:
		return audit.ResourceTypeBranchRule
	case protection.TypeTag:
		return audit.ResourceTypeTagRule
	case protection.TypePush:
		return audit.ResourceTypePushRule
	}
	return audit.ResourceTypeBranchRule
}

func (s *Service) getRuleUserAndUserGroups(
	ctx context.Context,
	rule *types.Rule,
) (
	map[int64]*types.PrincipalInfo, []int64,
	map[int64]*types.UserGroupInfo, []int64, //nolint:unparam
	error,
) {
	protection, err := s.parseRule(rule)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to parse rule: %w", err)
	}

	userMap, ruleUserIDs, err := s.getRuleUsers(ctx, protection)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to get rule users: %w", err)
	}
	userGroupMap, ruleGroupIDs, err := s.getRuleUserGroups(ctx, protection)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to get rule user groups: %w", err)
	}

	return userMap, ruleUserIDs, userGroupMap, ruleGroupIDs, nil
}

func (s *Service) getRuleUsers(
	ctx context.Context,
	protection protection.Protection,
) (map[int64]*types.PrincipalInfo, []int64, error) {
	ruleUserIDs, err := protection.UserIDs()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user IDs from rule: %w", err)
	}

	userMap, err := s.principalInfoCache.Map(ctx, ruleUserIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get principal infos: %w", err)
	}

	return userMap, ruleUserIDs, nil
}

func (s *Service) getRuleUserGroups(
	ctx context.Context,
	protection protection.Protection,
) (map[int64]*types.UserGroupInfo, []int64, error) {
	ruleGroupIDs, err := protection.UserGroupIDs()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get group IDs from rule: %w", err)
	}

	userGroupInfoMap := make(map[int64]*types.UserGroupInfo)

	if len(ruleGroupIDs) == 0 {
		return userGroupInfoMap, []int64{}, nil
	}

	groupMap, err := s.userGroupStore.Map(ctx, ruleGroupIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get userGroup infos: %w", err)
	}

	for k, v := range groupMap {
		userGroupInfoMap[k] = v.ToUserGroupInfo()
	}
	return userGroupInfoMap, ruleGroupIDs, nil
}

func (s *Service) parseRule(rule *types.Rule) (protection.Protection, error) {
	protection, err := s.protectionManager.FromJSON(rule.Type, rule.Definition, false)
	if err != nil {
		return nil, fmt.Errorf("failed to parse json rule definition: %w", err)
	}

	return protection, nil
}

func (s *Service) sendSSE(
	ctx context.Context,
	parentID int64,
	parentType enum.RuleParent,
	sseType enum.SSEType,
	rule *types.Rule,
) {
	spaceID := parentID
	if parentType == enum.RuleParentRepo {
		repo, err := s.repoStore.Find(ctx, parentID)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("failed to find repo")
			return
		}
		spaceID = repo.ParentID
	}
	s.sseStreamer.Publish(ctx, spaceID, sseType, rule)
}

// backfillRuleRepositories populates the rule's Repositories field with
// the repositories specified in the rule's RepoTarget.
func (s *Service) backfillRuleRepositories(
	ctx context.Context,
	rule *types.Rule,
) error {
	var repoTarget protection.RepoTarget
	err := json.Unmarshal(rule.RepoTarget, &repoTarget)
	if err != nil {
		return fmt.Errorf("failed to unmarshal rule.RepoTarget: %w", err)
	}

	ids := repoTarget.Include.IDs
	ids = append(ids, repoTarget.Exclude.IDs...)

	// Deduplicate IDs
	uniqueIDs := make(map[int64]struct{})
	for _, id := range ids {
		uniqueIDs[id] = struct{}{}
	}

	rule.Repositories = make(map[int64]*types.RepositoryCore, len(uniqueIDs))
	for repoID := range uniqueIDs {
		repo, err := s.repoIDCache.Get(ctx, repoID)
		if err != nil {
			if errors.Is(err, store.ErrResourceNotFound) {
				continue
			}
			return fmt.Errorf("failed to get repository from cache: %w", err)
		}
		rule.Repositories[repoID] = repo
	}

	return nil
}
