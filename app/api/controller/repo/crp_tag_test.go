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

package repo

import (
	"context"
	"testing"
	"time"

	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCrpCreateCommitTag(t *testing.T) {
	ctx := context.Background()
	created := &git.CreateCommitTagOutput{CommitTag: git.CommitTag{
		Name: "v1", SHA: crpTestSHA, IsAnnotated: true, Message: crpReleaseBranch,
	}}

	t.Run("target defaults and tagger comes from the session", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("CreateCommitTag", mock.Anything, mock.MatchedBy(func(p *git.CreateCommitTagParams) bool {
			return p.Name == "v1" && p.Target == crpDefaultBranch && p.Message == crpReleaseBranch &&
				p.Tagger != nil && p.Tagger.Email == crpAliceEmail && p.TaggerDate != nil &&
				time.Since(*p.TaggerDate) < time.Minute
		})).Return(created, nil)

		out, violations, err := e.c.CreateCommitTag(ctx, crpSession(), crpRepoRef,
			&CreateCommitTagInput{Name: "v1", Message: crpReleaseBranch})

		require.NoError(t, err)
		assert.Empty(t, violations)
		assert.Equal(t, "v1", out.Name)
		assert.True(t, out.IsAnnotated)
	})

	t.Run("dry run reports violations without touching git", func(t *testing.T) {
		e := crpNewEnv(t)
		e.ruleStore.rules = []types.RuleInfoInternal{crpTagRule(crpCreateForbidden)}

		out, _, err := e.c.CreateCommitTag(ctx, crpSession(), crpRepoRef,
			&CreateCommitTagInput{Name: "v1", DryRunRules: true})

		require.NoError(t, err)
		assert.True(t, out.DryRunRules)
		require.Len(t, out.RuleViolations, 1)
		e.git.AssertNotCalled(t, "CreateCommitTag", mock.Anything, mock.Anything)
	})

	t.Run("critical violation blocks creation", func(t *testing.T) {
		e := crpNewEnv(t)
		e.setAuthorizer(crpDenyPermAuthorizer{denied: map[enum.Permission]bool{enum.PermissionRepoEdit: true}})
		e.ruleStore.rules = []types.RuleInfoInternal{crpTagRule(crpCreateForbidden)}

		_, violations, err := e.c.CreateCommitTag(ctx, crpSession(), crpRepoRef, &CreateCommitTagInput{Name: "v1"})

		require.NoError(t, err)
		require.Len(t, violations, 1)
		e.git.AssertNotCalled(t, "CreateCommitTag", mock.Anything, mock.Anything)
	})

	t.Run("repo owner can bypass the violation", func(t *testing.T) {
		e := crpNewEnv(t)
		e.ruleStore.rules = []types.RuleInfoInternal{crpTagRule(crpCreateOwnerAllowed)}
		e.git.On("CreateCommitTag", mock.Anything, mock.Anything).Return(created, nil)

		out, violations, err := e.c.CreateCommitTag(ctx, crpSession(), crpRepoRef,
			&CreateCommitTagInput{Name: "v1", BypassRules: true})

		require.NoError(t, err)
		assert.Empty(t, violations)
		require.Len(t, out.RuleViolations, 1)
		assert.True(t, out.RuleViolations[0].Bypassed)
	})

	t.Run("rule store failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.ruleStore.err = errCrpBoom

		_, _, err := e.c.CreateCommitTag(ctx, crpSession(), crpRepoRef, &CreateCommitTagInput{Name: "v1"})

		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("git failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("CreateCommitTag", mock.Anything, mock.Anything).Return(nil, errCrpBoom)

		_, _, err := e.c.CreateCommitTag(ctx, crpSession(), crpRepoRef, &CreateCommitTagInput{Name: "v1"})

		require.ErrorIs(t, err, errCrpBoom)
	})
}

func TestCrpDeleteCommitTag(t *testing.T) {
	ctx := context.Background()

	t.Run("deletes the tag", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("DeleteTag", mock.Anything, mock.MatchedBy(func(p *git.DeleteTagParams) bool {
			return p.Name == "v1"
		})).Return(nil)

		out, violations, err := e.c.DeleteCommitTag(ctx, crpSession(), crpRepoRef, "v1", false, false)

		require.NoError(t, err)
		assert.Empty(t, violations)
		assert.Empty(t, out.RuleViolations)
		e.git.AssertNumberOfCalls(t, "DeleteTag", 1)
	})

	t.Run("dry run reports violations without touching git", func(t *testing.T) {
		e := crpNewEnv(t)
		e.ruleStore.rules = []types.RuleInfoInternal{crpTagRule(crpDeleteForbidden)}

		out, _, err := e.c.DeleteCommitTag(ctx, crpSession(), crpRepoRef, "v1", false, true)

		require.NoError(t, err)
		assert.True(t, out.DryRunRules)
		require.Len(t, out.RuleViolations, 1)
		e.git.AssertNotCalled(t, "DeleteTag", mock.Anything, mock.Anything)
	})

	t.Run("critical violation blocks deletion", func(t *testing.T) {
		e := crpNewEnv(t)
		e.setAuthorizer(crpDenyPermAuthorizer{denied: map[enum.Permission]bool{enum.PermissionRepoEdit: true}})
		e.ruleStore.rules = []types.RuleInfoInternal{crpTagRule(crpDeleteForbidden)}

		_, violations, err := e.c.DeleteCommitTag(ctx, crpSession(), crpRepoRef, "v1", false, false)

		require.NoError(t, err)
		require.Len(t, violations, 1)
		assert.Equal(t, "lifecycle.delete", violations[0].Violations[0].Code)
		e.git.AssertNotCalled(t, "DeleteTag", mock.Anything, mock.Anything)
	})

	t.Run("repo owner can bypass the violation", func(t *testing.T) {
		e := crpNewEnv(t)
		e.ruleStore.rules = []types.RuleInfoInternal{crpTagRule(crpDeleteOwnerAllowed)}
		e.git.On("DeleteTag", mock.Anything, mock.Anything).Return(nil)

		out, violations, err := e.c.DeleteCommitTag(ctx, crpSession(), crpRepoRef, "v1", true, false)

		require.NoError(t, err)
		assert.Empty(t, violations)
		require.Len(t, out.RuleViolations, 1)
		assert.True(t, out.RuleViolations[0].Bypassed)
	})

	t.Run("rule store failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.ruleStore.err = errCrpBoom

		_, _, err := e.c.DeleteCommitTag(ctx, crpSession(), crpRepoRef, "v1", false, false)

		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("git failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("DeleteTag", mock.Anything, mock.Anything).Return(errCrpBoom)

		_, _, err := e.c.DeleteCommitTag(ctx, crpSession(), crpRepoRef, "v1", false, false)

		require.ErrorIs(t, err, errCrpBoom)
	})
}

func TestCrpListCommitTags(t *testing.T) {
	ctx := context.Background()

	t.Run("maps tags including commits", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("ListCommitTags", mock.Anything, mock.MatchedBy(func(p *git.ListCommitTagsParams) bool {
			return p.IncludeCommit && p.Query == "v" && p.Page == 3 && p.PageSize == 5 &&
				p.Sort == git.TagSortOptionName && p.Order == git.SortOrderAsc
		})).Return(&git.ListCommitTagsOutput{Tags: []git.CommitTag{
			{Name: "v1", SHA: crpTestSHA, Commit: &git.Commit{SHA: crpTestSHAOther, Title: "first"}},
			{Name: "v2", SHA: crpTestSHAOther},
		}}, nil)

		tags, err := e.c.ListCommitTags(ctx, crpSession(), crpRepoRef, true, &types.TagFilter{
			Query: "v", Page: 3, Size: 5, Sort: enum.TagSortOptionName, Order: enum.OrderAsc,
		})

		require.NoError(t, err)
		require.Len(t, tags, 2)
		assert.Equal(t, "v1", tags[0].Name)
		require.NotNil(t, tags[0].Commit)
		assert.Equal(t, "first", tags[0].Commit.Title)
		assert.Nil(t, tags[1].Commit)
	})

	t.Run("git failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("ListCommitTags", mock.Anything, mock.Anything).Return(nil, errCrpBoom)

		_, err := e.c.ListCommitTags(ctx, crpSession(), crpRepoRef, false, &types.TagFilter{})

		require.ErrorIs(t, err, errCrpBoom)
	})
}
