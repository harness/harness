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
	"fmt"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

var RuleInfoFilterTypeBranch = func(r *types.RuleInfoInternal) (bool, error) {
	return r.Type == TypeBranch, nil
}

var RuleInfoFilterStatusActive = func(r *types.RuleInfoInternal) (bool, error) {
	return r.State == enum.RuleStateActive, nil
}

func GetBranchRuleInfos(
	protection BranchProtection,
	defaultBranch string,
	branchName string,
	filterFns ...func(*types.RuleInfoInternal) (bool, error),
) (ruleInfos []types.RuleInfo, err error) {
	v, ok := protection.(branchRuleSet)
	if !ok {
		return ruleInfos, nil
	}

	err = v.forEachRuleMatchBranch(
		defaultBranch,
		branchName,
		func(r *types.RuleInfoInternal, _ BranchProtection) error {
			for _, filterFn := range filterFns {
				allow, err := filterFn(r)
				if err != nil {
					return fmt.Errorf("rule info filter function error: %w", err)
				}

				if !allow {
					return nil
				}
			}

			ruleInfos = append(ruleInfos, r.RuleInfo)

			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process each rule in ruleSet: %w", err)
	}

	return ruleInfos, nil
}
