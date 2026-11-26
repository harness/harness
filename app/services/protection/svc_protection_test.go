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
	"slices"
	"testing"

	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/app/store"
	gitnesserrors "github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var svcRepoTarget = json.RawMessage(`{"include":{},"exclude":{}}`)

type svcRuleStore struct {
	store.RuleStore
	rules   []types.RuleInfoInternal
	err     error
	types   []enum.RuleType
	repoID  int64
	repoArg *types.RepositoryCore
}

func (s *svcRuleStore) ListAllRepoRules(
	_ context.Context, repoID int64, ruleTypes ...enum.RuleType,
) ([]types.RuleInfoInternal, error) {
	s.repoID = repoID
	s.types = ruleTypes
	return s.rules, s.err
}

func (s *svcRuleStore) ListOnlyRepoRules(
	_ context.Context, repo *types.RepositoryCore, ruleTypes ...enum.RuleType,
) ([]types.RuleInfoInternal, error) {
	s.repoArg = repo
	s.types = ruleTypes
	return s.rules, s.err
}

func svcManager(ruleStore store.RuleStore) *Manager {
	m := NewManager(ruleStore)
	_ = m.Register(TypeBranch, func() Definition { return &Branch{} })
	_ = m.Register(TypeTag, func() Definition { return &Tag{} })
	_ = m.Register(TypePush, func() Definition { return &Push{} })
	return m
}

func svcRule(id int64, ruleType enum.RuleType, definition string) types.RuleInfoInternal {
	return types.RuleInfoInternal{
		RuleInfo:   types.RuleInfo{ID: id, Type: ruleType, State: enum.RuleStateActive, Identifier: "rule"},
		RepoTarget: svcRepoTarget,
		Pattern:    json.RawMessage(`{}`),
		Definition: json.RawMessage(definition),
	}
}

func TestSvcManagerRegister(t *testing.T) {
	m := NewManager(nil)
	require.NoError(t, m.Register(TypeBranch, func() Definition { return &Branch{} }))
	require.ErrorIs(t, m.Register(TypeBranch, func() Definition { return &Branch{} }), ErrAlreadyRegistered)
}

func TestSvcManagerFromJSON(t *testing.T) {
	m := svcManager(nil)

	tests := []struct {
		name    string
		ruleTyp enum.RuleType
		raw     string
		params  sanitizeParams
		wantErr error
		anyErr  bool
	}{
		{"valid loose", TypeBranch, `{"bypass":{"user_ids":[1]}}`, SanitizeLoose(), nil, false},
		{"unknown type", enum.RuleType("bogus"), `{}`, SanitizeLoose(), ErrUnrecognizedType, false},
		{"unknown field is fine when loose", TypeBranch, `{"nope":1}`, SanitizeLoose(), nil, false},
		{
			"unknown field rejected when strict", TypeBranch, `{"nope":1}`,
			SanitizeStrictForParent(enum.RuleParentRepo), nil, true,
		},
		{"syntax error", TypeBranch, `{`, SanitizeLoose(), nil, true},
		{"sanitize failure", TypeBranch, `{"bypass":{"user_ids":[-1]}}`, SanitizeLoose(), nil, true},
		{
			"merge queue not allowed on space", TypeBranch,
			`{"pullreq":{"merge_queue":{"status_checks":{"require_identifiers":["a"]},"group_size":1,` +
				`"checks_concurrency":1,"max_check_duration_seconds":1}}}`,
			SanitizeStrictForParent(enum.RuleParentSpace), nil, true,
		},
		{
			"merge queue allowed on repo", TypeBranch,
			`{"pullreq":{"merge_queue":{"status_checks":{"require_identifiers":["a"]},"group_size":1,` +
				`"checks_concurrency":1,"max_check_duration_seconds":1}}}`,
			SanitizeStrictForParent(enum.RuleParentRepo), nil, false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, err := m.FromJSON(tt.ruleTyp, json.RawMessage(tt.raw), tt.params)
			switch {
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
			case tt.anyErr:
				require.Error(t, err)
			default:
				require.NoError(t, err)
				assert.NotNil(t, def)
			}
		})
	}
}

func TestSvcManagerSanitizeJSON(t *testing.T) {
	m := svcManager(nil)

	out, err := m.SanitizeJSON(TypeTag, json.RawMessage(`{"lifecycle":{"create_forbidden":true}}`), SanitizeLoose())
	require.NoError(t, err)
	assert.JSONEq(t, `{"bypass":{},"lifecycle":{"create_forbidden":true}}`, string(out))

	_, err = m.SanitizeJSON(enum.RuleType("bogus"), json.RawMessage(`{}`), SanitizeLoose())
	require.ErrorIs(t, err, ErrUnrecognizedType)
}

func TestSvcManagerListRules(t *testing.T) {
	ctx := context.Background()
	repo := &types.RepositoryCore{ID: 5, Path: "root/repo"}
	rules := []types.RuleInfoInternal{svcRule(1, TypeBranch, `{}`)}

	t.Run("list repo rules", func(t *testing.T) {
		rs := &svcRuleStore{rules: rules}
		got, err := svcManager(rs).ListRepoRules(ctx, 5, TypeBranch, TypeTag)
		require.NoError(t, err)
		assert.Equal(t, rules, got)
		assert.Equal(t, int64(5), rs.repoID)
		assert.Equal(t, []enum.RuleType{TypeBranch, TypeTag}, rs.types)
	})
	t.Run("list only repo rules", func(t *testing.T) {
		rs := &svcRuleStore{rules: rules}
		got, err := svcManager(rs).ListOnlyRepoRules(ctx, repo, TypeBranch)
		require.NoError(t, err)
		assert.Equal(t, rules, got)
		assert.Same(t, repo, rs.repoArg)
	})
	t.Run("typed rule sets", func(t *testing.T) {
		rs := &svcRuleStore{rules: rules}
		m := svcManager(rs)

		branch, err := m.ListRepoBranchRules(ctx, 5)
		require.NoError(t, err)
		assert.Equal(t, []enum.RuleType{TypeBranch}, rs.types)
		assert.IsType(t, branchRuleSet{}, branch)

		onlyBranch, err := m.ListOnlyRepoBranchRules(ctx, repo)
		require.NoError(t, err)
		assert.IsType(t, branchRuleSet{}, onlyBranch)

		tag, err := m.ListRepoTagRules(ctx, 5)
		require.NoError(t, err)
		assert.Equal(t, []enum.RuleType{TypeTag}, rs.types)
		assert.IsType(t, tagRuleSet{}, tag)

		push, err := m.ListRepoPushRules(ctx, 5)
		require.NoError(t, err)
		assert.Equal(t, []enum.RuleType{TypePush}, rs.types)
		assert.IsType(t, pushRuleSet{}, push)
	})
	t.Run("store failures are wrapped", func(t *testing.T) {
		boom := errors.New("db down")
		m := svcManager(&svcRuleStore{err: boom})

		_, err := m.ListRepoRules(ctx, 5)
		require.ErrorIs(t, err, boom)
		_, err = m.ListOnlyRepoRules(ctx, repo)
		require.ErrorIs(t, err, boom)
		_, err = m.ListRepoBranchRules(ctx, 5)
		require.ErrorIs(t, err, boom)
		_, err = m.ListOnlyRepoBranchRules(ctx, repo)
		require.ErrorIs(t, err, boom)
		_, err = m.ListRepoTagRules(ctx, 5)
		require.ErrorIs(t, err, boom)
		_, err = m.ListRepoPushRules(ctx, 5)
		require.ErrorIs(t, err, boom)
	})
}

func TestSvcIsBypassedAndIsCritical(t *testing.T) {
	violation := []types.Violation{{Code: "c"}}
	active := types.RuleInfo{State: enum.RuleStateActive}

	tests := []struct {
		name         string
		in           []types.RuleViolations
		wantBypassed bool
		wantCritical bool
	}{
		{"empty", nil, false, false},
		{"active violation", []types.RuleViolations{{Rule: active, Violations: violation}}, false, true},
		{
			"bypassed violation",
			[]types.RuleViolations{{Rule: active, Violations: violation, Bypassed: true}}, true, false,
		},
		{
			"monitor rule", []types.RuleViolations{
				{Rule: types.RuleInfo{State: enum.RuleStateMonitor}, Violations: violation},
			}, false, false,
		},
		{"no violations", []types.RuleViolations{{Rule: active, Bypassed: true}}, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantBypassed, IsBypassed(tt.in))
			assert.Equal(t, tt.wantCritical, IsCritical(tt.in))
		})
	}
}

func TestSvcPrintRuleScope(t *testing.T) {
	assert.Equal(t, `repository "a/b"`, printRuleScope(types.RuleInfo{RepoPath: "a/b"}))
	assert.Equal(t, `scope "a"`, printRuleScope(types.RuleInfo{SpacePath: "a"}))
	assert.Equal(t, "unknown scope", printRuleScope(types.RuleInfo{}))
}

func TestSvcPushRule_Violations(t *testing.T) {
	ctx := context.Background()
	oversize := &git.FindOversizeFilesOutput{TotalsPerLimit: map[int64]int64{100: 2, 50: 1}}

	tests := []struct {
		name      string
		rule      Push
		in        PushViolationsInput
		wantCodes []string
	}{
		{
			name: "no violations without input",
			rule: Push{Push: DefPush{FileSizeLimit: 100, PrincipalCommitterMatch: true, SecretScanningEnabled: true}},
			in:   PushViolationsInput{},
		},
		{
			name:      "file size limit",
			rule:      Push{Push: DefPush{FileSizeLimit: 100}},
			in:        PushViolationsInput{FindOversizeFilesOutput: oversize},
			wantCodes: []string{codePushFileSizeLimit},
		},
		{
			name: "file size limit disabled",
			rule: Push{Push: DefPush{FileSizeLimit: 0}},
			in:   PushViolationsInput{FindOversizeFilesOutput: oversize},
		},
		{
			name: "oversize files below the limit are ignored",
			rule: Push{Push: DefPush{FileSizeLimit: 1000}},
			in:   PushViolationsInput{FindOversizeFilesOutput: oversize},
		},
		{
			name:      "committer mismatch",
			rule:      Push{Push: DefPush{PrincipalCommitterMatch: true}},
			in:        PushViolationsInput{PrincipalCommitterMatch: true, CommitterMismatchCount: 2},
			wantCodes: []string{codePushPrincipalCommitterMatch},
		},
		{
			name: "committer mismatch ignored when rule disabled",
			rule: Push{},
			in:   PushViolationsInput{PrincipalCommitterMatch: true, CommitterMismatchCount: 2},
		},
		{
			name:      "secrets",
			rule:      Push{Push: DefPush{SecretScanningEnabled: true}},
			in:        PushViolationsInput{SecretScanningEnabled: true, FoundSecretsCount: 3},
			wantCodes: []string{codeSecretScanningEnabled},
		},
		{
			name: "all violations",
			rule: Push{Push: DefPush{
				FileSizeLimit: 10, PrincipalCommitterMatch: true, SecretScanningEnabled: true,
			}},
			in: PushViolationsInput{
				FindOversizeFilesOutput: oversize,
				PrincipalCommitterMatch: true, CommitterMismatchCount: 1,
				SecretScanningEnabled: true, FoundSecretsCount: 1,
			},
			wantCodes: []string{codePushFileSizeLimit, codePushPrincipalCommitterMatch, codeSecretScanningEnabled},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := tt.rule.Violations(ctx, &tt.in)
			require.NoError(t, err)
			require.Len(t, out.Violations, 1)

			var codes []string
			for _, v := range out.Violations[0].Violations {
				codes = append(codes, v.Code)
			}
			assert.Equal(t, tt.wantCodes, codes)
		})
	}

	t.Run("bypass flags", func(t *testing.T) {
		rule := Push{Bypass: DefBypass{UserIDs: []int64{7}}, Push: DefPush{SecretScanningEnabled: true}}
		in := PushViolationsInput{
			Actor: &types.Principal{ID: 7}, AllowBypass: true, SecretScanningEnabled: true, FoundSecretsCount: 1,
		}
		out, err := rule.Violations(ctx, &in)
		require.NoError(t, err)
		assert.True(t, out.Violations[0].Bypassable)
		assert.True(t, out.Violations[0].Bypassed)

		in.AllowBypass = false
		out, err = rule.Violations(ctx, &in)
		require.NoError(t, err)
		assert.True(t, out.Violations[0].Bypassable)
		assert.False(t, out.Violations[0].Bypassed)

		in.Actor = &types.Principal{ID: 8}
		out, err = rule.Violations(ctx, &in)
		require.NoError(t, err)
		assert.False(t, out.Violations[0].Bypassable)
	})
}

func TestSvcPushRule_PushVerify(t *testing.T) {
	ctx := context.Background()
	rule := Push{
		Bypass: DefBypass{RepoOwners: true},
		Push:   DefPush{FileSizeLimit: 42, PrincipalCommitterMatch: true, SecretScanningEnabled: true},
	}

	out, violations, err := rule.PushVerify(ctx, PushVerifyInput{IsRepoOwner: true, Actor: &types.Principal{ID: 1}})
	require.NoError(t, err)
	assert.Empty(t, violations)
	assert.Equal(t, []int64{42}, out.FileSizeLimits)
	assert.True(t, out.PrincipalCommitterMatch)
	assert.True(t, out.SecretScanningEnabled)
}

func TestSvcPushRule_Misc(t *testing.T) {
	rule := Push{Bypass: DefBypass{UserIDs: []int64{1, 2}, UserGroupIDs: []int64{3}}}

	ids, err := rule.UserIDs()
	require.NoError(t, err)
	assert.Equal(t, []int64{1, 2}, ids)

	gids, err := rule.UserGroupIDs()
	require.NoError(t, err)
	assert.Equal(t, []int64{3}, gids)

	require.NoError(t, rule.Sanitize())
	require.Error(t, (&Push{Bypass: DefBypass{UserIDs: []int64{0}}}).Sanitize())

	require.NoError(t, rule.SupportsParent(enum.RuleParentSpace))
	require.NoError(t, rule.SupportsParent(enum.RuleParentRepo))
}

func TestSvcPushRuleSet(t *testing.T) {
	ctx := context.Background()
	m := svcManager(nil)

	rules := []types.RuleInfoInternal{
		svcRule(1, TypePush, `{"push":{"file_size_limit":10,"secret_scanning_enabled":true}}`),
		svcRule(2, TypePush, `{"push":{"file_size_limit":20,"principal_committer_match":true}}`),
	}
	notMatching := svcRule(3, TypePush, `{"push":{"file_size_limit":30}}`)
	notMatching.RepoTarget = json.RawMessage(`{"include":{"ids":[999]},"exclude":{}}`)
	rules = append(rules, notMatching)

	set := m.FilterPushProtection(append(rules, svcRule(4, TypeBranch, `{}`)))

	out, violations, err := set.PushVerify(ctx, PushVerifyInput{RepoID: 1, RepoPath: "root/repo"})
	require.NoError(t, err)
	assert.Empty(t, violations)
	assert.Equal(t, []int64{10, 20}, out.FileSizeLimits)
	assert.True(t, out.PrincipalCommitterMatch)
	assert.True(t, out.SecretScanningEnabled)
	assert.Len(t, out.Protections, 2)
	assert.Contains(t, out.Protections, int64(1))
	assert.NotContains(t, out.Protections, int64(3))

	t.Run("violations are back filled with rule info", func(t *testing.T) {
		vOut, err := set.Violations(ctx, &PushViolationsInput{
			Protections: out.Protections, SecretScanningEnabled: true, FoundSecretsCount: 1,
		})
		require.NoError(t, err)
		require.Len(t, vOut.Violations, 2)
		assert.Equal(t, int64(1), vOut.Violations[0].Rule.ID)
		assert.Equal(t, int64(2), vOut.Violations[1].Rule.ID)
		assert.NotEmpty(t, vOut.Violations[0].Violations)
		assert.Empty(t, vOut.Violations[1].Violations)

		empty, err := set.Violations(ctx, &PushViolationsInput{})
		require.NoError(t, err)
		assert.Empty(t, empty.Violations)
	})
	t.Run("collects user and group ids", func(t *testing.T) {
		withIDs := m.FilterPushProtection([]types.RuleInfoInternal{
			svcRule(1, TypePush, `{"bypass":{"user_ids":[1,2],"user_group_ids":[5]}}`),
			svcRule(2, TypePush, `{"bypass":{"user_ids":[2,3]}}`),
		})
		ids, err := withIDs.UserIDs()
		require.NoError(t, err)
		slices.Sort(ids)
		assert.Equal(t, []int64{1, 2, 3}, ids)

		gids, err := withIDs.UserGroupIDs()
		require.NoError(t, err)
		assert.Equal(t, []int64{5}, gids)
	})
	t.Run("bad rule definition", func(t *testing.T) {
		bad := m.FilterPushProtection([]types.RuleInfoInternal{svcRule(1, TypePush, `{`)})
		_, _, err := bad.PushVerify(ctx, PushVerifyInput{RepoID: 1})
		require.Error(t, err)

		_, err = bad.UserIDs()
		require.Error(t, err)
		_, err = bad.UserGroupIDs()
		require.Error(t, err)
	})
	t.Run("bad repo target", func(t *testing.T) {
		r := svcRule(1, TypePush, `{}`)
		r.RepoTarget = json.RawMessage(`{`)
		_, _, err := m.FilterPushProtection([]types.RuleInfoInternal{r}).PushVerify(ctx, PushVerifyInput{})
		require.Error(t, err)
	})
}

func TestSvcPushVerifyInput_HasViolations(t *testing.T) {
	tests := []struct {
		name string
		in   PushViolationsInput
		want bool
	}{
		{"nothing", PushViolationsInput{}, false},
		{"empty oversize output", PushViolationsInput{FindOversizeFilesOutput: &git.FindOversizeFilesOutput{}}, false},
		{"oversize files", PushViolationsInput{FindOversizeFilesOutput: &git.FindOversizeFilesOutput{
			FileInfosPerLimit: map[int64][]git.FileInfo{10: {{}}},
		}}, true},
		{"committer mismatch", PushViolationsInput{CommitterMismatchCount: 1}, true},
		{"secrets", PushViolationsInput{FoundSecretsCount: 1}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.in.HasViolations())
		})
	}
}

func TestSvcTagRule(t *testing.T) {
	rule := Tag{Bypass: DefBypass{UserIDs: []int64{1}, UserGroupIDs: []int64{2}}}

	gids, err := rule.UserGroupIDs()
	require.NoError(t, err)
	assert.Equal(t, []int64{2}, gids)

	require.NoError(t, rule.Sanitize())
	require.Error(t, (&Tag{Bypass: DefBypass{UserIDs: []int64{-1}}}).Sanitize())
	require.NoError(t, rule.SupportsParent(enum.RuleParentSpace))

	t.Run("ref change verify ignores non tag refs", func(t *testing.T) {
		out, err := rule.RefChangeVerify(context.Background(), RefChangeVerifyInput{RefType: RefTypeBranch})
		require.NoError(t, err)
		assert.Empty(t, out)
	})

	t.Run("tag rule set collects user groups", func(t *testing.T) {
		set := svcManager(nil).FilterTagProtection([]types.RuleInfoInternal{
			svcRule(1, TypeTag, `{"bypass":{"user_group_ids":[4,5]}}`),
			svcRule(2, TypeBranch, `{"bypass":{"user_group_ids":[6]}}`),
		})
		got, err := set.UserGroupIDs()
		require.NoError(t, err)
		slices.Sort(got)
		assert.Equal(t, []int64{4, 5}, got)
	})
}

func TestSvcBranchRule_Sanitize(t *testing.T) {
	tests := []struct {
		name    string
		rule    Branch
		wantErr bool
	}{
		{"empty", Branch{}, false},
		{"bad bypass", Branch{Bypass: DefBypass{UserIDs: []int64{0}}}, true},
		{"bad pull req", Branch{PullReq: DefPullReq{Approvals: DefApprovals{RequireMinimumCount: -1}}}, true},
		{"valid", Branch{
			Bypass:  DefBypass{UserIDs: []int64{1}},
			PullReq: DefPullReq{Approvals: DefApprovals{RequireMinimumCount: 1}},
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.Sanitize()
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestSvcBranchRule_CreatePullReqVerify(t *testing.T) {
	ctx := context.Background()
	rule := Branch{
		Bypass: DefBypass{UserIDs: []int64{9}},
		PullReq: DefPullReq{Reviewers: DefReviewers{
			RequestCodeOwners: true, DefaultReviewerIDs: []int64{1, 2}, DefaultUserGroupReviewerIDs: []int64{3},
		}},
	}

	out, violations, err := rule.CreatePullReqVerify(ctx, CreatePullReqVerifyInput{Actor: &types.Principal{ID: 9}})
	require.NoError(t, err)
	assert.Empty(t, violations)
	assert.True(t, out.RequestCodeOwners)
	assert.Equal(t, []int64{1, 2}, out.DefaultReviewerIDs)
	assert.Equal(t, []int64{3}, out.DefaultGroupReviewerIDs)

	ids, err := rule.UserIDs()
	require.NoError(t, err)
	slices.Sort(ids)
	assert.Equal(t, []int64{1, 2, 9}, ids)

	gids, err := rule.UserGroupIDs()
	require.NoError(t, err)
	assert.Equal(t, []int64{3}, gids)
}

func TestSvcBranchRule_RefChangeVerify(t *testing.T) {
	ctx := context.Background()
	repo := &types.RepositoryCore{ID: 1, Path: "root/repo"}

	mq := &DefMergeQueue{
		StatusChecks: DefStatusChecks{RequireIdentifiers: []string{"ci"}}, GroupSize: 1, ChecksConcurrency: 1,
		MaxCheckDurationSeconds: 60,
	}

	tests := []struct {
		name      string
		rule      Branch
		in        RefChangeVerifyInput
		wantCodes []string
		bypassed  bool
	}{
		{
			name: "non branch ref type is ignored",
			rule: Branch{Lifecycle: DefBranchLifecycle{DefLifecycle: DefLifecycle{DeleteForbidden: true}}},
			in:   RefChangeVerifyInput{RefType: RefTypeTag, RefAction: RefActionDelete, RefNames: []string{"a"}, Repo: repo},
		},
		{
			name: "no ref names are ignored",
			rule: Branch{Lifecycle: DefBranchLifecycle{DefLifecycle: DefLifecycle{DeleteForbidden: true}}},
			in:   RefChangeVerifyInput{RefType: RefTypeBranch, RefAction: RefActionDelete, Repo: repo},
		},
		{
			name: "delete forbidden",
			rule: Branch{Lifecycle: DefBranchLifecycle{DefLifecycle: DefLifecycle{DeleteForbidden: true}}},
			in: RefChangeVerifyInput{
				RefType: RefTypeBranch, RefAction: RefActionDelete, RefNames: []string{"a"}, Repo: repo,
			},
			wantCodes: []string{codeLifecycleDelete},
		},
		{
			name: "bypassed by listed user",
			rule: Branch{
				Bypass:    DefBypass{UserIDs: []int64{5}},
				Lifecycle: DefBranchLifecycle{DefLifecycle: DefLifecycle{DeleteForbidden: true}},
			},
			in: RefChangeVerifyInput{
				RefType: RefTypeBranch, RefAction: RefActionDelete, RefNames: []string{"a"}, Repo: repo,
				Actor: &types.Principal{ID: 5}, AllowBypass: true,
			},
			wantCodes: []string{codeLifecycleDelete},
			bypassed:  true,
		},
		{
			name: "merge queue blocks direct updates",
			rule: Branch{PullReq: DefPullReq{MergeQueue: mq}},
			in: RefChangeVerifyInput{
				RefType: RefTypeBranch, RefAction: RefActionUpdate, RefNames: []string{"main", "dev"}, Repo: repo,
			},
			wantCodes: []string{codeMergeQueueBranchUpdateVerify, codeMergeQueueBranchUpdateVerify},
		},
		{
			name: "merge queue does not block creation",
			rule: Branch{PullReq: DefPullReq{MergeQueue: mq}},
			in: RefChangeVerifyInput{
				RefType: RefTypeBranch, RefAction: RefActionCreate, RefNames: []string{"main"}, Repo: repo,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations, err := tt.rule.RefChangeVerify(ctx, tt.in)
			require.NoError(t, err)

			var codes []string
			for _, v := range violations {
				for _, vv := range v.Violations {
					codes = append(codes, vv.Code)
				}
				assert.Equal(t, tt.bypassed, v.Bypassed)
			}
			assert.Equal(t, tt.wantCodes, codes)
		})
	}
}

func TestSvcBranchRule_SupportsParent(t *testing.T) {
	plain := Branch{}
	withMQ := Branch{PullReq: DefPullReq{MergeQueue: &DefMergeQueue{}}}

	require.NoError(t, plain.SupportsParent(enum.RuleParentSpace))
	require.NoError(t, withMQ.SupportsParent(enum.RuleParentRepo))

	err := withMQ.SupportsParent(enum.RuleParentSpace)
	require.Error(t, err)
	assert.Equal(t, gitnesserrors.StatusInvalidArgument, gitnesserrors.AsStatus(err))
}

func TestSvcBranchRuleSet(t *testing.T) {
	ctx := context.Background()
	m := svcManager(nil)

	rules := []types.RuleInfoInternal{
		svcRule(1, TypeBranch, `{"pullreq":{"reviewers":{"request_code_owners":true,`+
			`"default_reviewer_ids":[1,2],"default_user_group_reviewer_ids":[7]}}}`),
		svcRule(2, TypeBranch, `{"pullreq":{"reviewers":{"default_reviewer_ids":[2,3],`+
			`"default_user_group_reviewer_ids":[7,8]}},"bypass":{"user_group_ids":[9]}}`),
	}
	nonMatching := svcRule(3, TypeBranch, `{"pullreq":{"reviewers":{"default_reviewer_ids":[99]}}}`)
	nonMatching.Pattern = json.RawMessage(`{"include":["release-*"]}`)
	rules = append(rules, nonMatching)

	set := m.FilterBranchProtection(rules)

	t.Run("create pull request verify combines outputs", func(t *testing.T) {
		out, violations, err := set.CreatePullReqVerify(ctx, CreatePullReqVerifyInput{
			RepoID: 1, RepoPath: "root/repo", DefaultBranch: "main", TargetBranch: "main",
		})
		require.NoError(t, err)
		assert.Empty(t, violations)
		assert.True(t, out.RequestCodeOwners)
		assert.Equal(t, []int64{1, 2, 3}, out.DefaultReviewerIDs)
		assert.Equal(t, []int64{7, 8}, out.DefaultGroupReviewerIDs)
	})
	t.Run("pattern matches only release branches", func(t *testing.T) {
		out, _, err := set.CreatePullReqVerify(ctx, CreatePullReqVerifyInput{
			RepoID: 1, RepoPath: "root/repo", DefaultBranch: "main", TargetBranch: "release-1",
		})
		require.NoError(t, err)
		assert.Contains(t, out.DefaultReviewerIDs, int64(99))
	})
	t.Run("collects user group ids", func(t *testing.T) {
		gids, err := set.UserGroupIDs()
		require.NoError(t, err)
		slices.Sort(gids)
		assert.Equal(t, []int64{7, 8, 9}, gids)
	})
	t.Run("ref change verify back fills rules", func(t *testing.T) {
		locked := m.FilterBranchProtection([]types.RuleInfoInternal{
			svcRule(5, TypeBranch, `{"lifecycle":{"delete_forbidden":true}}`),
		})
		violations, err := locked.RefChangeVerify(context.Background(), RefChangeVerifyInput{
			Repo: &types.RepositoryCore{ID: 1, Path: "root/repo"}, RefType: RefTypeBranch,
			RefAction: RefActionDelete, RefNames: []string{"x"},
		})
		require.NoError(t, err)
		require.Len(t, violations, 1)
		assert.Equal(t, int64(5), violations[0].Rule.ID)
	})
	t.Run("broken rule is reported", func(t *testing.T) {
		broken := m.FilterBranchProtection([]types.RuleInfoInternal{svcRule(1, TypeBranch, `{`)})
		_, _, err := broken.CreatePullReqVerify(ctx, CreatePullReqVerifyInput{RepoID: 1})
		require.Error(t, err)
		_, err = broken.RefChangeVerify(ctx, RefChangeVerifyInput{
			Repo: &types.RepositoryCore{ID: 1}, RefNames: []string{"a"},
		})
		require.Error(t, err)
		_, err = broken.UserGroupIDs()
		require.Error(t, err)
	})
}

func TestSvcGetBranchRuleInfos(t *testing.T) {
	m := svcManager(nil)
	disabled := svcRule(2, TypeBranch, `{}`)
	disabled.State = enum.RuleStateDisabled
	set := m.FilterBranchProtection([]types.RuleInfoInternal{svcRule(1, TypeBranch, `{}`), disabled})

	all, err := GetBranchRuleInfos(1, "root/repo", set, "main", "main")
	require.NoError(t, err)
	assert.Len(t, all, 2)

	active, err := GetBranchRuleInfos(1, "root/repo", set, "main", "main", RuleInfoFilterStatusActive)
	require.NoError(t, err)
	require.Len(t, active, 1)
	assert.Equal(t, int64(1), active[0].ID)

	tagsOnly, err := GetBranchRuleInfos(1, "root/repo", set, "main", "main",
		func(r *types.RuleInfoInternal) (bool, error) { return r.Type == TypeTag, nil })
	require.NoError(t, err)
	assert.Empty(t, tagsOnly)

	_, err = GetBranchRuleInfos(1, "root/repo", set, "main", "main",
		func(*types.RuleInfoInternal) (bool, error) { return false, errors.New("boom") })
	require.Error(t, err)

	// protection that is not a branch rule set yields nothing.
	none, err := GetBranchRuleInfos(1, "root/repo", &Branch{}, "main", "main")
	require.NoError(t, err)
	assert.Empty(t, none)

	typeFiltered, err := GetBranchRuleInfos(1, "root/repo", set, "main", "main", RuleInfoFilterTypeBranch)
	require.NoError(t, err)
	assert.Len(t, typeFiltered, 2)
}

func TestSvcDeduplicateInt64Slice(t *testing.T) {
	assert.Equal(t, []int64{}, deduplicateInt64Slice(nil))
	assert.Equal(t, []int64{3, 1, 2}, deduplicateInt64Slice([]int64{3, 1, 3, 2, 1}))
}

func TestSvcDefBypass(t *testing.T) {
	ctx := context.Background()
	actor := &types.Principal{ID: 4}

	resolver := func(ids ...int64) func(context.Context, []int64) ([]int64, error) {
		return func(context.Context, []int64) ([]int64, error) { return ids, nil }
	}

	tests := []struct {
		name        string
		bypass      DefBypass
		actor       *types.Principal
		isRepoOwner bool
		resolver    func(context.Context, []int64) ([]int64, error)
		want        bool
	}{
		{"nil actor", DefBypass{UserIDs: []int64{4}}, nil, false, nil, false},
		{"listed user", DefBypass{UserIDs: []int64{4}}, actor, false, nil, true},
		{"unlisted user", DefBypass{UserIDs: []int64{5}}, actor, false, nil, false},
		{"repo owner", DefBypass{RepoOwners: true}, actor, true, nil, true},
		{"not repo owner", DefBypass{RepoOwners: true}, actor, false, nil, false},
		{"member of listed group", DefBypass{UserGroupIDs: []int64{1}}, actor, false, resolver(4), true},
		{"not a member of listed group", DefBypass{UserGroupIDs: []int64{1}}, actor, false, resolver(5), false},
		{
			"resolver failure denies bypass", DefBypass{UserIDs: []int64{4}}, actor, false,
			func(context.Context, []int64) ([]int64, error) { return nil, errors.New("boom") }, false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.bypass.matches(ctx, tt.actor, tt.isRepoOwner, tt.resolver))
		})
	}

	require.NoError(t, DefBypass{UserIDs: []int64{1}}.Sanitize())
	require.Error(t, DefBypass{UserIDs: []int64{0}}.Sanitize())
}

func TestSvcValidators(t *testing.T) {
	require.NoError(t, validateIDSlice(nil))
	require.NoError(t, validateIDSlice([]int64{1, 2}))
	require.Error(t, validateIDSlice([]int64{0}))
	require.Error(t, validateIDSlice(make([]int64, maxElements+1)))

	require.NoError(t, validateIdentifierSlice([]string{"a"}))
	require.Error(t, validateIdentifierSlice([]string{"a", ""}))
	require.Error(t, validateIdentifierSlice(make([]string, maxElements+1)))
}

func TestSvcPatternAndRepoTargetValidate(t *testing.T) {
	p := &Pattern{Include: []string{"feat-*"}, Exclude: []string{"tmp/**"}}
	assert.JSONEq(t, `{"include":["feat-*"],"exclude":["tmp/**"]}`, string(p.JSON()))
	require.NoError(t, p.Validate())
	require.ErrorIs(t, (&Pattern{Include: []string{""}}).Validate(), ErrPatternEmpty)
	require.ErrorIs(t, (&Pattern{Exclude: []string{"[a-"}}).Validate(), ErrInvalidGlobstarPattern)

	rt := &RepoTarget{
		Include: RepoTargetFilter{IDs: []int64{1}, Patterns: []string{"a*"}},
		Exclude: RepoTargetFilter{Patterns: []string{"b*"}},
	}
	assert.JSONEq(t, `{"include":{"ids":[1],"patterns":["a*"]},"exclude":{"patterns":["b*"]}}`, string(rt.JSON()))
	require.NoError(t, rt.Validate())

	tests := []struct {
		name string
		rt   RepoTarget
	}{
		{"bad include id", RepoTarget{Include: RepoTargetFilter{IDs: []int64{0}}}},
		{"bad exclude id", RepoTarget{Exclude: RepoTargetFilter{IDs: []int64{-1}}}},
		{"bad include pattern", RepoTarget{Include: RepoTargetFilter{Patterns: []string{""}}}},
		{"bad exclude pattern", RepoTarget{Exclude: RepoTargetFilter{Patterns: []string{"[a-"}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Error(t, tt.rt.Validate())
		})
	}
}

func TestSvcMatchesRefAndRepoParseErrors(t *testing.T) {
	_, err := matchesRef(json.RawMessage(`{`), "main", "main")
	require.Error(t, err)
	_, err = matchesRefs(json.RawMessage(`{`), "main", "main")
	require.Error(t, err)
	_, err = matchesRepo(json.RawMessage(`{`), 1, "a/b", "")
	require.Error(t, err)

	ok, err := matchesRef(json.RawMessage(`{"default":true}`), "main", "main")
	require.NoError(t, err)
	assert.True(t, ok)
	ok, err = matchesRef(json.RawMessage(`{"default":true}`), "main", "dev")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestSvcDefPullReqSanitizers(t *testing.T) {
	t.Run("approvals", func(t *testing.T) {
		tests := []struct {
			name    string
			in      DefApprovals
			wantErr bool
		}{
			{"zero", DefApprovals{}, false},
			{"negative count", DefApprovals{RequireMinimumCount: -1}, true},
			{"latest commit alone", DefApprovals{RequireLatestCommit: true}, true},
			{"latest commit with code owners", DefApprovals{RequireLatestCommit: true, RequireCodeOwners: true}, false},
			{"latest commit with count", DefApprovals{RequireLatestCommit: true, RequireMinimumCount: 1}, false},
			{
				"latest commit with default reviewers",
				DefApprovals{RequireLatestCommit: true, RequireMinimumDefaultReviewerCount: 1}, false,
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.Equal(t, tt.wantErr, tt.in.Sanitize() != nil)
			})
		}
	})
	t.Run("merge", func(t *testing.T) {
		m := DefMerge{StrategiesAllowed: []enum.MergeMethod{enum.MergeMethodSquash, enum.MergeMethodMerge}}
		require.NoError(t, m.Sanitize())
		assert.True(t, slices.IsSorted(m.StrategiesAllowed))

		require.Error(t, (&DefMerge{StrategiesAllowed: []enum.MergeMethod{"bogus"}}).Sanitize())
		require.Error(t, (&DefMerge{
			StrategiesAllowed: []enum.MergeMethod{enum.MergeMethodSquash, enum.MergeMethodSquash},
		}).Sanitize())
	})
	t.Run("reviewers", func(t *testing.T) {
		require.NoError(t, (&DefReviewers{DefaultReviewerIDs: []int64{1}}).Sanitize())
		require.Error(t, (&DefReviewers{DefaultReviewerIDs: []int64{0}}).Sanitize())
		require.Error(t, (&DefReviewers{DefaultUserGroupReviewerIDs: []int64{0}}).Sanitize())
	})
	t.Run("status checks", func(t *testing.T) {
		require.NoError(t, (&DefStatusChecks{RequireIdentifiers: []string{"a"}}).Sanitize())
		require.Error(t, (&DefStatusChecks{RequireIdentifiers: []string{""}}).Sanitize())
	})
	t.Run("merge queue", func(t *testing.T) {
		valid := func() *DefMergeQueue {
			return &DefMergeQueue{
				StatusChecks: DefStatusChecks{RequireIdentifiers: []string{"ci"}},
				GroupSize:    2, ChecksConcurrency: 2, MaxCheckDurationSeconds: 30,
			}
		}
		var nilQueue *DefMergeQueue
		require.NoError(t, nilQueue.Sanitize())
		require.NoError(t, valid().Sanitize())

		mutations := map[string]func(q *DefMergeQueue){
			"no checks":         func(q *DefMergeQueue) { q.StatusChecks.RequireIdentifiers = nil },
			"zero group":        func(q *DefMergeQueue) { q.GroupSize = 0 },
			"big group":         func(q *DefMergeQueue) { q.GroupSize = MaxGroupSize + 1 },
			"zero concurrency":  func(q *DefMergeQueue) { q.ChecksConcurrency = 0 },
			"big concurrency":   func(q *DefMergeQueue) { q.ChecksConcurrency = MaxChecksConcurrency + 1 },
			"zero duration":     func(q *DefMergeQueue) { q.MaxCheckDurationSeconds = 0 },
			"empty check ident": func(q *DefMergeQueue) { q.StatusChecks.RequireIdentifiers = []string{""} },
		}
		for name, mutate := range mutations {
			t.Run(name, func(t *testing.T) {
				q := valid()
				mutate(q)
				require.Error(t, q.Sanitize())
			})
		}

		setup, err := valid().GetMergeQueueSetup(MergeQueueSetupInput{})
		require.NoError(t, err)
		assert.True(t, setup.IsActive())
		assert.Equal(t, 2, setup.GroupSize)
		assert.False(t, MergeQueueSetup{}.IsActive())

		emptySetup, err := nilQueue.GetMergeQueueSetup(MergeQueueSetupInput{})
		require.NoError(t, err)
		assert.False(t, emptySetup.IsActive())

		vs, err := valid().MergeQueueBranchUpdateVerify(MergeQueueBranchUpdateInput{TargetBranch: "main"})
		require.NoError(t, err)
		require.Len(t, vs, 1)
		vs, err = nilQueue.MergeQueueBranchUpdateVerify(MergeQueueBranchUpdateInput{})
		require.NoError(t, err)
		assert.Empty(t, vs)
	})
	t.Run("pull request definition reports the failing section", func(t *testing.T) {
		tests := map[string]DefPullReq{
			"approvals": {Approvals: DefApprovals{RequireMinimumCount: -1}},
			"status":    {StatusChecks: DefStatusChecks{RequireIdentifiers: []string{""}}},
			"merge":     {Merge: DefMerge{StrategiesAllowed: []enum.MergeMethod{"bogus"}}},
			"reviewers": {Reviewers: DefReviewers{DefaultReviewerIDs: []int64{0}}},
			"queue":     {MergeQueue: &DefMergeQueue{}},
		}
		for name, def := range tests {
			t.Run(name, func(t *testing.T) {
				require.Error(t, def.Sanitize())
			})
		}
		require.NoError(t, (&DefPullReq{}).Sanitize())
		require.NoError(t, DefComments{}.Sanitize())
		require.NoError(t, (&DefCommits{}).Sanitize())
	})
}

func TestSvcDefPullReq_RequiredChecksAndCreate(t *testing.T) {
	ctx := context.Background()
	def := &DefPullReq{
		StatusChecks: DefStatusChecks{RequireIdentifiers: []string{"a", "b"}},
		Reviewers:    DefReviewers{RequestCodeOwners: true, DefaultReviewerIDs: []int64{5}},
	}

	out, err := def.RequiredChecks(ctx, RequiredChecksInput{})
	require.NoError(t, err)
	assert.Len(t, out.RequiredIdentifiers, 2)

	cOut, vs, err := def.CreatePullReqVerify(ctx, CreatePullReqVerifyInput{})
	require.NoError(t, err)
	assert.Empty(t, vs)
	assert.True(t, cOut.RequestCodeOwners)
	assert.Equal(t, []int64{5}, cOut.DefaultReviewerIDs)
}

func TestSvcDefStatusChecksJSON(t *testing.T) {
	raw, err := json.Marshal(DefStatusChecks{RequireIdentifiers: []string{"ci"}})
	require.NoError(t, err)
	assert.JSONEq(t, `{"require_identifiers":["ci"],"require_uids":["ci"]}`, string(raw))

	var fromNew DefStatusChecks
	require.NoError(t, json.Unmarshal([]byte(`{"require_identifiers":["new"],"require_uids":["old"]}`), &fromNew))
	assert.Equal(t, []string{"new"}, fromNew.RequireIdentifiers)

	var fromLegacy DefStatusChecks
	require.NoError(t, json.Unmarshal([]byte(`{"require_uids":["old"]}`), &fromLegacy))
	assert.Equal(t, []string{"old"}, fromLegacy.RequireIdentifiers)

	require.Error(t, json.Unmarshal([]byte(`{"require_identifiers":5}`), &DefStatusChecks{}))
}

func TestSvcGetCodeOwnerApprovalStatus(t *testing.T) {
	approved := codeowners.UserEvaluation{ReviewDecision: enum.PullReqReviewDecisionApproved}
	changeReq := codeowners.UserEvaluation{ReviewDecision: enum.PullReqReviewDecisionChangeReq}
	pending := codeowners.UserEvaluation{ReviewDecision: enum.PullReqReviewDecisionPending}

	tests := []struct {
		name         string
		entry        codeowners.EvaluationEntry
		want         enum.PullReqReviewDecision
		wantApprover int
	}{
		{"nothing", codeowners.EvaluationEntry{}, enum.PullReqReviewDecisionPending, 0},
		{
			"pending only",
			codeowners.EvaluationEntry{UserEvaluations: []codeowners.UserEvaluation{pending}},
			enum.PullReqReviewDecisionPending, 0,
		},
		{
			"one approval",
			codeowners.EvaluationEntry{UserEvaluations: []codeowners.UserEvaluation{approved, pending}},
			enum.PullReqReviewDecisionApproved, 1,
		},
		{
			"change request beats approval",
			codeowners.EvaluationEntry{UserEvaluations: []codeowners.UserEvaluation{approved, changeReq}},
			enum.PullReqReviewDecisionChangeReq, 0,
		},
		{
			"group approval",
			codeowners.EvaluationEntry{UserGroupEvaluations: []codeowners.UserGroupEvaluation{
				{Evaluations: []codeowners.UserEvaluation{approved, approved}},
			}},
			enum.PullReqReviewDecisionApproved, 2,
		},
		{
			"group change request beats user approval",
			codeowners.EvaluationEntry{
				UserEvaluations: []codeowners.UserEvaluation{approved},
				UserGroupEvaluations: []codeowners.UserGroupEvaluation{
					{Evaluations: []codeowners.UserEvaluation{changeReq}},
				},
			},
			enum.PullReqReviewDecisionChangeReq, 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision, approvers := getCodeOwnerApprovalStatus(tt.entry)
			assert.Equal(t, tt.want, decision)
			assert.Len(t, approvers, tt.wantApprover)
		})
	}
}
