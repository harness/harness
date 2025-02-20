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

const TypeBranch types.RuleType = "branch"

// Branch implements protection rules for the rule type TypeBranch.
type Branch struct {
	Bypass    DefBypass    `json:"bypass"`
	PullReq   DefPullReq   `json:"pullreq"`
	Lifecycle DefLifecycle `json:"lifecycle"`
}

var (
	// ensures that the Branch type implements Definition interface.
	_ Definition = (*Branch)(nil)
	_ Protection = (*Branch)(nil)
)

func (v *Branch) MergeVerify(
	ctx context.Context,
	in MergeVerifyInput,
) (out MergeVerifyOutput, violations []types.RuleViolations, err error) {
	out, violations, err = v.PullReq.MergeVerify(ctx, in)
	if err != nil {
		return out, violations, fmt.Errorf("merge verify error: %w", err)
	}

	bypassable := v.Bypass.matches(ctx, in.Actor, in.IsRepoOwner, in.ResolveUserGroupID)
	bypassed := in.AllowBypass && bypassable
	for i := range violations {
		violations[i].Bypassable = bypassable
		violations[i].Bypassed = bypassed
	}

	return
}

func (v *Branch) RequiredChecks(
	ctx context.Context,
	in RequiredChecksInput,
) (RequiredChecksOutput, error) {
	out, err := v.PullReq.RequiredChecks(ctx, in)
	if err != nil {
		return RequiredChecksOutput{}, err
	}

	ids := out.RequiredIdentifiers
	if len(ids) == 0 {
		return RequiredChecksOutput{}, nil
	}

	var (
		requiredIDs   map[string]struct{}
		bypassableIDs map[string]struct{}
	)

	if bypassable := v.Bypass.matches(ctx, in.Actor, in.IsRepoOwner, in.ResolveUserGroupID); bypassable {
		bypassableIDs = ids
	} else {
		requiredIDs = ids
	}

	return RequiredChecksOutput{
		RequiredIdentifiers:   requiredIDs,
		BypassableIdentifiers: bypassableIDs,
	}, nil
}

func (v *Branch) CreatePullReqVerify(
	ctx context.Context,
	in CreatePullReqVerifyInput,
) (CreatePullReqVerifyOutput, []types.RuleViolations, error) {
	var out CreatePullReqVerifyOutput

	out, violations, err := v.PullReq.CreatePullReqVerify(ctx, in)
	if err != nil {
		return CreatePullReqVerifyOutput{}, nil, err
	}

	bypassable := v.Bypass.matches(ctx, in.Actor, in.IsRepoOwner, in.ResolveUserGroupID)
	bypassed := in.AllowBypass && bypassable
	for i := range violations {
		violations[i].Bypassable = bypassable
		violations[i].Bypassed = bypassed
	}

	return out, violations, nil
}

func (v *Branch) RefChangeVerify(
	ctx context.Context,
	in RefChangeVerifyInput,
) (violations []types.RuleViolations, err error) {
	if in.RefType != RefTypeBranch || len(in.RefNames) == 0 {
		return []types.RuleViolations{}, nil
	}

	violations, err = v.Lifecycle.RefChangeVerify(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("lifecycle error: %w", err)
	}

	bypassable := v.Bypass.matches(ctx, in.Actor, in.IsRepoOwner, in.ResolveUserGroupID)
	bypassed := in.AllowBypass && bypassable
	for i := range violations {
		violations[i].Bypassable = bypassable
		violations[i].Bypassed = bypassed
	}

	return
}

func (v *Branch) UserIDs() ([]int64, error) {
	return v.Bypass.UserIDs, nil
}

func (v *Branch) UserGroupIDs() ([]int64, error) {
	return v.Bypass.UserGroupIDs, nil
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
