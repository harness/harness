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

	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (s *Service) getRuleUserAndUserGroups(
	ctx context.Context,
	rule *types.Rule,
) (map[int64]*types.PrincipalInfo, map[int64]*types.UserGroupInfo, error) {
	protection, err := s.parseRule(rule)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse rule: %w", err)
	}

	userMap, err := s.getRuleUsers(ctx, protection)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get rule users: %w", err)
	}
	userGroupMap, err := s.getRuleUserGroups(ctx, protection)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get rule user groups: %w", err)
	}

	return userMap, userGroupMap, nil
}

func (s *Service) getRuleUsers(
	ctx context.Context,
	protection protection.Protection,
) (map[int64]*types.PrincipalInfo, error) {
	userIDs, err := protection.UserIDs()
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID from rule: %w", err)
	}

	userMap, err := s.principalInfoCache.Map(ctx, userIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get principal infos: %w", err)
	}

	return userMap, nil
}

func (s *Service) getRuleUserGroups(
	ctx context.Context,
	protection protection.Protection,
) (map[int64]*types.UserGroupInfo, error) {
	groupIDs, err := protection.UserGroupIDs()
	if err != nil {
		return nil, fmt.Errorf("failed to get group IDs from rule: %w", err)
	}

	userGroupInfoMap := make(map[int64]*types.UserGroupInfo)

	if len(groupIDs) == 0 {
		return userGroupInfoMap, nil
	}

	groupMap, err := s.userGroupStore.Map(ctx, groupIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get userGroup infos: %w", err)
	}

	for k, v := range groupMap {
		userGroupInfoMap[k] = v.ToUserGroupInfo()
	}
	return userGroupInfoMap, nil
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
