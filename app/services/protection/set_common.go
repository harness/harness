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

	"golang.org/x/exp/maps"
)

func forEachRule(
	manager *Manager,
	rules []types.RuleInfoInternal,
	fn func(r *types.RuleInfoInternal, p Protection) error,
) error {
	for i := range rules {
		r := rules[i]

		protection, err := manager.FromJSON(r.Type, r.Definition, false)
		if err != nil {
			return fmt.Errorf("forEachRule: failed to parse protection definition ID=%d Type=%s: %w",
				r.ID, r.Type, err)
		}

		err = fn(&r, protection)
		if err != nil {
			return fmt.Errorf("forEachRule: failed to process rule ID=%d Type=%s: %w",
				r.ID, r.Type, err)
		}
	}

	return nil
}

func collectIDs(
	manager *Manager,
	rules []types.RuleInfoInternal,
	extract func(Protection) ([]int64, error),
) ([]int64, error) {
	mapIDs := make(map[int64]bool)

	err := forEachRule(manager, rules, func(_ *types.RuleInfoInternal, p Protection) error {
		ids, err := extract(p)
		if err != nil {
			return fmt.Errorf("failed to extract IDs: %w", err)
		}

		for _, id := range ids {
			mapIDs[id] = true
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to process each rule in ruleSet: %w", err)
	}

	return maps.Keys(mapIDs), nil
}

func refChangeVerifyFunc(
	ctx context.Context,
	in RefChangeVerifyInput,
	violations *[]types.RuleViolations,
) func(r *types.RuleInfoInternal, p Protection, matched []string) error {
	return func(r *types.RuleInfoInternal, p Protection, matched []string) error {
		ruleIn := in
		ruleIn.RefNames = matched

		rVs, err := p.RefChangeVerify(ctx, ruleIn)
		if err != nil {
			return err
		}

		*violations = append(*violations, backFillRule(rVs, r.RuleInfo)...)
		return nil
	}
}

func forEachRuleMatchRefs(
	manager *Manager,
	rules []types.RuleInfoInternal,
	defaultBranch string,
	refNames []string,
	fn func(r *types.RuleInfoInternal, p Protection, matched []string) error,
) error {
	for i := range rules {
		r := rules[i]

		matched, err := matchedNames(r.Pattern, defaultBranch, refNames...)
		if err != nil {
			return err
		}
		if len(matched) == 0 {
			continue
		}

		protection, err := manager.FromJSON(r.Type, r.Definition, false)
		if err != nil {
			return fmt.Errorf("forEachRuleMatchRefs: failed to parse protection definition ID=%d Type=%s: %w",
				r.ID, r.Type, err)
		}

		err = fn(&r, protection, matched)
		if err != nil {
			return fmt.Errorf("forEachRuleMatchRefs: failed to process rule ID=%d Type=%s: %w",
				r.ID, r.Type, err)
		}
	}

	return nil
}
