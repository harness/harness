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
	"github.com/harness/gitness/types/enum"
)

const TypeTag enum.RuleType = "tag"

// Tag implements protection rules for the rule type TypeTag.
type Tag struct {
	Bypass    DefBypass       `json:"bypass"`
	Lifecycle DefTagLifecycle `json:"lifecycle"`
}

var (
	// ensures that the Branch type implements Definition interface.
	_ Definition    = (*Tag)(nil)
	_ TagProtection = (*Tag)(nil)
)

func (t *Tag) RefChangeVerify(
	ctx context.Context,
	in RefChangeVerifyInput,
) ([]types.RuleViolations, error) {
	if in.RefType != RefTypeTag || len(in.RefNames) == 0 {
		return []types.RuleViolations{}, nil
	}

	violations, err := t.Lifecycle.RefChangeVerify(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("lifecycle error: %w", err)
	}

	bypassable := t.Bypass.matches(ctx, in.Actor, in.IsRepoOwner, in.ResolveUserGroupID)
	bypassed := in.AllowBypass && bypassable
	for i := range violations {
		violations[i].Bypassable = bypassable
		violations[i].Bypassed = bypassed
	}

	return violations, nil
}

func (t *Tag) UserIDs() ([]int64, error) {
	return t.Bypass.UserIDs, nil
}

func (t *Tag) UserGroupIDs() ([]int64, error) {
	return t.Bypass.UserGroupIDs, nil
}

func (t *Tag) Sanitize() error {
	if err := t.Bypass.Sanitize(); err != nil {
		return fmt.Errorf("bypass: %w", err)
	}

	if err := t.Lifecycle.Sanitize(); err != nil {
		return fmt.Errorf("lifecycle: %w", err)
	}

	return nil
}
