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

package protection

import (
	"context"
	"fmt"

	"github.com/harness/gitness/types"
)

type tagRuleSet struct {
	rules   []types.RuleInfoInternal
	manager *Manager
}

var _ Protection = tagRuleSet{} // ensure that ruleSet implements the Protection interface.

func (s tagRuleSet) RefChangeVerify(ctx context.Context, in RefChangeVerifyInput) ([]types.RuleViolations, error) {
	var violations []types.RuleViolations

	err := forEachRuleMatchRefs(
		s.manager,
		s.rules,
		in.Repo.ID,
		in.Repo.Identifier,
		"",
		in.RefNames,
		refChangeVerifyFunc(ctx, in, &violations),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process each rule in ruleSet: %w", err)
	}

	return violations, nil
}

func (s tagRuleSet) UserIDs() ([]int64, error) {
	return collectIDs(s.manager, s.rules, Protection.UserIDs)
}

func (s tagRuleSet) UserGroupIDs() ([]int64, error) {
	return collectIDs(s.manager, s.rules, Protection.UserGroupIDs)
}
