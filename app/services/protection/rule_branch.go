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

	"golang.org/x/exp/slices"
)

var TypeBranch types.RuleType = "branch"

// Branch implements protection rules for the rule type TypeBranch.
type Branch struct {
	Bypass    DefBypass    `json:"bypass"`
	PullReq   DefPullReq   `json:"pullreq"`
	Lifecycle DefLifecycle `json:"lifecycle"`
}

var (
	// ensures that the Branch type implements Definition interface.
	_ Definition = (*Branch)(nil)
)

//nolint:gocognit // well aware of this
func (v *Branch) MergeVerify(
	ctx context.Context,
	in MergeVerifyInput,
) (MergeVerifyOutput, []types.RuleViolations, error) {
	if v.isBypassed(in.Actor, in.IsSpaceOwner) {
		return MergeVerifyOutput{}, nil, nil
	}

	return v.PullReq.MergeVerify(ctx, in)
}

func (v *Branch) RefChangeVerify(
	ctx context.Context,
	in RefChangeVerifyInput,
) ([]types.RuleViolations, error) {
	if v.isBypassed(in.Actor, in.IsSpaceOwner) || in.RefType != RefTypeBranch || len(in.RefNames) == 0 {
		return nil, nil
	}

	return v.Lifecycle.RefChangeVerify(ctx, in)
}

func (v *Branch) Sanitize() error {
	if err := v.Bypass.Sanitize(); err != nil {
		return fmt.Errorf("bypass: %w", err)
	}

	if err := v.PullReq.Sanitize(); err != nil {
		return fmt.Errorf("pull request: %w", err)
	}

	if err := v.Lifecycle.Sanitize(); err != nil {
		return fmt.Errorf("lifecycle: %w", err)
	}

	return nil
}

func (v *Branch) isBypassed(actor *types.Principal, isSpaceOwner bool) bool {
	return actor != nil &&
		(actor.Admin ||
			v.Bypass.SpaceOwners && isSpaceOwner ||
			slices.Contains(v.Bypass.UserIDs, actor.ID))
}
