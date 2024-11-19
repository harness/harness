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

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// Find returns the protection rule by identifier.
func (s *Service) Find(ctx context.Context,
	parentType enum.RuleParent,
	parentID int64,
	identifier string,
) (*types.Rule, error) {
	rule, err := s.ruleStore.FindByIdentifier(ctx, parentType, parentID, identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to find protection rule by identifier: %w", err)
	}

	userMap, userGroupMap, err := s.getRuleUserAndUserGroups(ctx, rule)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule users and user groups: %w", err)
	}

	rule.Users = userMap
	rule.UserGroups = userGroupMap

	return rule, nil
}
