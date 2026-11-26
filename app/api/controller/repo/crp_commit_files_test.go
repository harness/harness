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
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func crpCommitOpts() *CommitFilesOptions {
	return &CommitFilesOptions{
		Title:  "add file",
		Branch: crpFeatureBranch,
		Actions: []CommitFileAction{
			{Action: git.CreateAction, Path: "a.txt", Payload: "hello"},
		},
	}
}

func TestCrpCommitFiles(t *testing.T) {
	ctx := context.Background()
	committed := git.CommitFilesResponse{
		CommitID:     crpTestSHA,
		ChangedFiles: []git.FileReference{{Path: "a.txt", SHA: crpTestSHAOther}},
	}

	newEnv := func(t *testing.T) *crpEnv {
		t.Helper()
		e := crpNewEnv(t)
		e.withMergeQueue(ctx, t, &crpMQEntryStore{})

		return e
	}

	t.Run("invalid options", func(t *testing.T) {
		e := newEnv(t)

		opts := crpCommitOpts()
		opts.NewBranch = "bad..branch"

		_, _, err := e.c.CommitFiles(ctx, crpSession(), crpRepoRef, opts)

		require.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, crpUserErrStatus(t, err))
	})

	t.Run("decodes payloads and returns changed files", func(t *testing.T) {
		e := newEnv(t)
		opts := crpCommitOpts()
		opts.Actions = append(opts.Actions, CommitFileAction{
			Action: git.CreateAction, Path: "b.bin", Encoding: enum.ContentEncodingTypeBase64,
			Payload: base64.StdEncoding.EncodeToString([]byte("binary")),
		})
		e.git.On("CommitFiles", mock.Anything, mock.MatchedBy(func(p *git.CommitFilesParams) bool {
			return p.Branch == crpFeatureBranch && len(p.Actions) == 2 &&
				string(p.Actions[0].Payload) == "hello" && string(p.Actions[1].Payload) == "binary" &&
				p.Author != nil && p.Author.Email == crpAliceEmail && p.Committer != nil
		})).Return(committed, nil)

		out, violations, err := e.c.CommitFiles(ctx, crpSession(), crpRepoRef, opts)

		require.NoError(t, err)
		assert.Empty(t, violations)
		assert.Equal(t, crpTestSHA, out.CommitID)
		assert.Equal(t, []types.FileReference{{Path: "a.txt", SHA: crpTestSHAOther}}, out.ChangedFiles)
	})

	t.Run("explicit author is kept", func(t *testing.T) {
		e := newEnv(t)
		opts := crpCommitOpts()
		opts.Author = &git.Identity{Name: "Bob", Email: "bob@example.com"}
		e.git.On("CommitFiles", mock.Anything, mock.MatchedBy(func(p *git.CommitFilesParams) bool {
			return p.Author.Email == "bob@example.com"
		})).Return(committed, nil)

		_, _, err := e.c.CommitFiles(ctx, crpSession(), crpRepoRef, opts)

		require.NoError(t, err)
	})

	t.Run("bypass rules uses bypass write params", func(t *testing.T) {
		e := newEnv(t)
		opts := crpCommitOpts()
		opts.BypassRules = true
		e.ruleStore.rules = []types.RuleInfoInternal{crpBranchRule(
			`{"bypass":{"repo_owners":true},"lifecycle":{"update_forbidden":true}}`)}
		e.git.On("CommitFiles", mock.Anything, mock.Anything).Return(committed, nil)

		out, violations, err := e.c.CommitFiles(ctx, crpSession(), crpRepoRef, opts)

		require.NoError(t, err)
		assert.Empty(t, violations)
		require.Len(t, out.RuleViolations, 1)
		assert.True(t, out.RuleViolations[0].Bypassed)
	})

	t.Run("new branch is verified as a branch creation", func(t *testing.T) {
		e := newEnv(t)
		e.setAuthorizer(crpDenyPermAuthorizer{denied: map[enum.Permission]bool{enum.PermissionRepoEdit: true}})
		e.ruleStore.rules = []types.RuleInfoInternal{crpBranchRule(crpCreateForbidden)}
		opts := crpCommitOpts()
		opts.NewBranch = "created"

		_, violations, err := e.c.CommitFiles(ctx, crpSession(), crpRepoRef, opts)

		require.NoError(t, err)
		require.Len(t, violations, 1)
		assert.Equal(t, "lifecycle.create", violations[0].Violations[0].Code)
		e.git.AssertNotCalled(t, "CommitFiles", mock.Anything, mock.Anything)
	})

	t.Run("dry run reports violations without touching git", func(t *testing.T) {
		e := newEnv(t)
		e.ruleStore.rules = []types.RuleInfoInternal{crpBranchRule(crpUpdateForbidden)}
		opts := crpCommitOpts()
		opts.DryRunRules = true

		out, _, err := e.c.CommitFiles(ctx, crpSession(), crpRepoRef, opts)

		require.NoError(t, err)
		assert.True(t, out.DryRunRules)
		require.Len(t, out.RuleViolations, 1)
		e.git.AssertNotCalled(t, "CommitFiles", mock.Anything, mock.Anything)
	})

	t.Run("queued branch blocks the commit", func(t *testing.T) {
		e := crpNewEnv(t)
		e.withMergeQueue(ctx, t, &crpMQEntryStore{queued: map[string]struct{}{crpFeatureBranch: {}}})

		_, violations, err := e.c.CommitFiles(ctx, crpSession(), crpRepoRef, crpCommitOpts())

		require.NoError(t, err)
		require.Len(t, violations, 1)
		e.git.AssertNotCalled(t, "CommitFiles", mock.Anything, mock.Anything)
	})

	t.Run("merge queue failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.withMergeQueue(ctx, t, &crpMQEntryStore{err: errCrpBoom})

		_, _, err := e.c.CommitFiles(ctx, crpSession(), crpRepoRef, crpCommitOpts())

		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("rule store failure", func(t *testing.T) {
		e := newEnv(t)
		e.ruleStore.err = errCrpBoom

		_, _, err := e.c.CommitFiles(ctx, crpSession(), crpRepoRef, crpCommitOpts())

		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("invalid base64 payload", func(t *testing.T) {
		e := newEnv(t)
		opts := crpCommitOpts()
		opts.Actions[0].Encoding = enum.ContentEncodingTypeBase64
		opts.Actions[0].Payload = "***not base64***"

		_, _, err := e.c.CommitFiles(ctx, crpSession(), crpRepoRef, opts)

		require.Error(t, err)
		assert.True(t, errors.IsInternal(err))
	})

	t.Run("push rule violations from the hook are surfaced", func(t *testing.T) {
		e := newEnv(t)
		hookErr := errors.Forbidden("blocked by push rules").SetDetails(map[string]any{
			hook.RuleViolationsErrorDetailsKey: json.RawMessage(
				`[{"rule":{"identifier":"push-rule"},"violations":[{"code":"c","message":"m"}]}]`),
		})
		e.git.On("CommitFiles", mock.Anything, mock.Anything).Return(git.CommitFilesResponse{}, hookErr)

		_, violations, err := e.c.CommitFiles(ctx, crpSession(), crpRepoRef, crpCommitOpts())

		require.NoError(t, err)
		require.Len(t, violations, 1)
		assert.Equal(t, "push-rule", violations[0].Rule.Identifier)
	})

	t.Run("git failure", func(t *testing.T) {
		e := newEnv(t)
		e.git.On("CommitFiles", mock.Anything, mock.Anything).Return(git.CommitFilesResponse{}, errCrpBoom)

		_, violations, err := e.c.CommitFiles(ctx, crpSession(), crpRepoRef, crpCommitOpts())

		require.ErrorIs(t, err, errCrpBoom)
		assert.Empty(t, violations)
	})
}
