// Copyright 2026 Harness, Inc.
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
	"encoding/json"
	"errors"
	"testing"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

var errSv5xBoom = errors.New("boom")

// sv5xFailingRule is a rule definition whose every verification fails.
type sv5xFailingRule struct{}

func (sv5xFailingRule) UserIDs() ([]int64, error)            { return nil, errSv5xBoom }
func (sv5xFailingRule) UserGroupIDs() ([]int64, error)       { return nil, errSv5xBoom }
func (sv5xFailingRule) Sanitize() error                      { return nil }
func (sv5xFailingRule) SupportsParent(enum.RuleParent) error { return nil }

func (sv5xFailingRule) RefChangeVerify(context.Context, RefChangeVerifyInput) ([]types.RuleViolations, error) {
	return nil, errSv5xBoom
}

func (sv5xFailingRule) MergeVerify(
	context.Context, MergeVerifyInput,
) (MergeVerifyOutput, []types.RuleViolations, error) {
	return MergeVerifyOutput{}, nil, errSv5xBoom
}

func (sv5xFailingRule) RequiredChecks(context.Context, RequiredChecksInput) (RequiredChecksOutput, error) {
	return RequiredChecksOutput{}, errSv5xBoom
}

func (sv5xFailingRule) MergeQueueBranchUpdateVerify(MergeQueueBranchUpdateInput) ([]types.RuleViolations, error) {
	return nil, errSv5xBoom
}

func (sv5xFailingRule) GetMergeQueueSetup(MergeQueueSetupInput) (MergeQueueSetup, error) {
	return MergeQueueSetup{}, errSv5xBoom
}

func (sv5xFailingRule) GetDeleteSourceBranch(DeleteSourceBranchInput) (bool, error) {
	return false, errSv5xBoom
}

func (sv5xFailingRule) CreatePullReqVerify(
	context.Context, CreatePullReqVerifyInput,
) (CreatePullReqVerifyOutput, []types.RuleViolations, error) {
	return CreatePullReqVerifyOutput{}, nil, errSv5xBoom
}

func (sv5xFailingRule) PushVerify(context.Context, PushVerifyInput) (PushVerifyOutput, []types.RuleViolations, error) {
	return PushVerifyOutput{}, nil, errSv5xBoom
}

func sv5xManager(t *testing.T, ruleType enum.RuleType) *Manager {
	t.Helper()
	m := NewManager(nil)
	require.NoError(t, m.Register(ruleType, func() Definition { return &sv5xFailingRule{} }))
	return m
}

func sv5xRules(ruleType enum.RuleType) []types.RuleInfoInternal {
	return []types.RuleInfoInternal{{
		RuleInfo:   types.RuleInfo{ID: 3, Identifier: "failing", Type: ruleType, State: enum.RuleStateActive},
		RepoTarget: json.RawMessage(`{}`),
		Pattern:    json.RawMessage(`{"default":true}`),
		Definition: json.RawMessage(`{}`),
	}}
}

func TestSv5xBranchRuleSetPropagatesRuleErrors(t *testing.T) {
	ctx := context.Background()
	set := branchRuleSet{rules: sv5xRules(TypeBranch), manager: sv5xManager(t, TypeBranch)}
	repo := &types.RepositoryCore{ID: 1, Path: "space/repo", DefaultBranch: "main"}
	pr := &types.PullReq{TargetBranch: "main"}

	_, _, err := set.MergeVerify(ctx, MergeVerifyInput{TargetRepo: repo, PullReq: pr})
	require.ErrorIs(t, err, errSv5xBoom)

	_, err = set.RequiredChecks(ctx, RequiredChecksInput{Repo: repo, PullReq: pr})
	require.ErrorIs(t, err, errSv5xBoom)

	_, err = set.GetMergeQueueSetup(MergeQueueSetupInput{Repo: repo, TargetBranch: "main"})
	require.ErrorIs(t, err, errSv5xBoom)

	_, err = set.GetDeleteSourceBranch(DeleteSourceBranchInput{Repo: repo, TargetBranch: "main"})
	require.ErrorIs(t, err, errSv5xBoom)

	_, err = set.MergeQueueBranchUpdateVerify(MergeQueueBranchUpdateInput{Repo: repo, TargetBranch: "main"})
	require.ErrorIs(t, err, errSv5xBoom)

	_, _, err = set.CreatePullReqVerify(ctx, CreatePullReqVerifyInput{
		RepoID: 1, RepoPath: "space/repo", DefaultBranch: "main", TargetBranch: "main",
	})
	require.ErrorIs(t, err, errSv5xBoom)

	_, err = set.RefChangeVerify(ctx, RefChangeVerifyInput{Repo: repo, RefNames: []string{"main"}})
	require.ErrorIs(t, err, errSv5xBoom)

	_, err = set.UserIDs()
	require.ErrorIs(t, err, errSv5xBoom)
	_, err = set.UserGroupIDs()
	require.ErrorIs(t, err, errSv5xBoom)
}

func TestSv5xTagRuleSetPropagatesRuleErrors(t *testing.T) {
	ctx := context.Background()
	rules := sv5xRules(TypeTag)
	rules[0].Pattern = json.RawMessage(`{"include":["**"]}`)
	set := tagRuleSet{rules: rules, manager: sv5xManager(t, TypeTag)}

	_, err := set.RefChangeVerify(ctx, RefChangeVerifyInput{
		Repo: &types.RepositoryCore{ID: 1, Path: "space/repo"}, RefNames: []string{"v1"},
	})
	require.ErrorIs(t, err, errSv5xBoom)

	_, err = set.UserIDs()
	require.ErrorIs(t, err, errSv5xBoom)
	_, err = set.UserGroupIDs()
	require.ErrorIs(t, err, errSv5xBoom)
}

func TestSv5xPushRuleSetPropagatesRuleErrors(t *testing.T) {
	set := pushRuleSet{rules: sv5xRules(TypePush), manager: sv5xManager(t, TypePush)}

	_, _, err := set.PushVerify(context.Background(), PushVerifyInput{RepoID: 1, RepoPath: "space/repo"})
	require.ErrorIs(t, err, errSv5xBoom)

	_, err = set.UserIDs()
	require.ErrorIs(t, err, errSv5xBoom)
}

func TestSv5xRuleSetsRejectUnexpectedDefinitionTypes(t *testing.T) {
	// A definition that is none of the protection interfaces other than Definition.
	manager := NewManager(nil)
	require.NoError(t, manager.Register(TypeBranch, func() Definition { return &sv5xPlainRule{} }))
	require.NoError(t, manager.Register(TypePush, func() Definition { return &sv5xPlainRule{} }))

	repo := &types.RepositoryCore{ID: 1, Path: "space/repo", DefaultBranch: "main"}

	branch := branchRuleSet{rules: sv5xRules(TypeBranch), manager: manager}
	_, err := branch.GetDeleteSourceBranch(DeleteSourceBranchInput{Repo: repo, TargetBranch: "main"})
	require.ErrorContains(t, err, "expected BranchProtection")

	_, err = branch.RefChangeVerify(context.Background(), RefChangeVerifyInput{Repo: repo, RefNames: []string{"main"}})
	require.ErrorContains(t, err, "expected RefProtection")

	push := pushRuleSet{rules: sv5xRules(TypePush), manager: manager}
	_, _, err = push.PushVerify(context.Background(), PushVerifyInput{RepoID: 1, RepoPath: "space/repo"})
	require.ErrorContains(t, err, "expected PushProtection")
}

func (sv5xFailingRule) Violations(context.Context, *PushViolationsInput) (PushViolationsOutput, error) {
	return PushViolationsOutput{}, errSv5xBoom
}

type sv5xPlainRule struct{}

func (sv5xPlainRule) UserIDs() ([]int64, error)            { return nil, nil }
func (sv5xPlainRule) UserGroupIDs() ([]int64, error)       { return nil, nil }
func (sv5xPlainRule) Sanitize() error                      { return nil }
func (sv5xPlainRule) SupportsParent(enum.RuleParent) error { return nil }
