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

package codeowners

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/harness/gitness/app/services/usergroup"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

var (
	errSvbBoom = errors.New("boom")

	svbFileSHA = sha.Must("1111111111111111111111111111111111111111")
)

type svbGit struct {
	git.Interface

	content  string
	size     int64
	mode     git.TreeNodeMode
	treeErr  error
	diffErr  error
	files    []string
	treeCall int
}

func (g *svbGit) GetTreeNode(_ context.Context, _ *git.GetTreeNodeParams) (*git.GetTreeNodeOutput, error) {
	g.treeCall++
	if g.treeErr != nil {
		return nil, g.treeErr
	}
	mode := g.mode
	if mode == "" {
		mode = git.TreeNodeModeFile
	}
	return &git.GetTreeNodeOutput{Node: git.TreeNode{Mode: mode, SHA: "filesha", Path: "CODEOWNERS"}}, nil
}

func (g *svbGit) GetBlob(_ context.Context, _ *git.GetBlobParams) (*git.GetBlobOutput, error) {
	size := g.size
	if size == 0 {
		size = int64(len(g.content))
	}
	return &git.GetBlobOutput{SHA: svbFileSHA, Size: size, Content: io.NopCloser(strings.NewReader(g.content))}, nil
}

func (g *svbGit) DiffFileNames(context.Context, *git.DiffParams) (git.DiffFileNamesOutput, error) {
	return git.DiffFileNamesOutput{Files: g.files}, g.diffErr
}

type svbPrincipalStore struct {
	store.PrincipalStore

	byEmail map[string]*types.Principal
	byUID   map[string]*types.Principal
	err     error
}

func (s *svbPrincipalStore) FindByEmail(_ context.Context, email string) (*types.Principal, error) {
	if s.err != nil {
		return nil, s.err
	}
	p, ok := s.byEmail[email]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return p, nil
}

func (s *svbPrincipalStore) FindManyByUID(_ context.Context, uids []string) ([]*types.Principal, error) {
	if s.err != nil {
		return nil, s.err
	}
	var out []*types.Principal
	for _, uid := range uids {
		if p, ok := s.byUID[uid]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

type svbResolver struct {
	groups map[string]*types.UserGroup
	err    error
}

func (r svbResolver) Resolve(_ context.Context, id string) (*types.UserGroup, error) {
	if r.err != nil {
		return nil, r.err
	}
	g, ok := r.groups[id]
	if !ok {
		return nil, usergroup.ErrNotFound
	}
	return g, nil
}

func svbRepo() *types.RepositoryCore {
	return &types.RepositoryCore{GitUID: "uid", DefaultBranch: "main"}
}

func svbService(g *svbGit, ps *svbPrincipalStore, r svbResolver) *Service {
	return New(nil, g, Config{FilePaths: []string{"CODEOWNERS"}}, ps, r)
}

func TestSvbErrorTypes(t *testing.T) {
	tooLarge := &TooLargeError{FileSize: 5 * oneMegabyte}
	require.Contains(t, tooLarge.Error(), "5.00MB")
	require.ErrorIs(t, tooLarge, &TooLargeError{})
	require.NotErrorIs(t, errSvbBoom, &TooLargeError{})

	parseErr := &FileParseError{LineNumber: 3, Line: "x", Err: errSvbBoom}
	require.Contains(t, parseErr.Error(), "line 3")
	require.ErrorIs(t, parseErr, errSvbBoom)
	require.ErrorIs(t, parseErr, &FileParseError{})
	require.NotErrorIs(t, errSvbBoom, &FileParseError{})

	invalid := &InvalidFileTypeError{Mode: git.TreeNodeModeSymlink}
	require.Contains(t, invalid.Error(), "regular file is required")
	require.ErrorIs(t, invalid, &InvalidFileTypeError{})
	require.NotErrorIs(t, errSvbBoom, &InvalidFileTypeError{})
}

func TestSvbEntryIsOwnershipReset(t *testing.T) {
	require.True(t, (&Entry{Pattern: "*"}).IsOwnershipReset())
	require.False(t, (&Entry{Pattern: "*", Owners: []string{"a@b.c"}}).IsOwnershipReset())
}

func TestSvbParseUserGroupOwner(t *testing.T) {
	id, ok := ParseUserGroupOwner("@org/team")
	require.True(t, ok)
	require.Equal(t, "org/team", id)
	_, ok = ParseUserGroupOwner("a@b.c")
	require.False(t, ok)
}

func TestSvbFindReviewerInList(t *testing.T) {
	reviewers := []*types.PullReqReviewer{
		{Reviewer: types.PrincipalInfo{ID: 1, UID: "u1", Email: "One@x.io"}},
		{Reviewer: types.PrincipalInfo{ID: 2, UID: "u2", Email: "two@x.io"}},
	}
	require.Equal(t, int64(1), findReviewerInList("one@x.io", "", reviewers).Reviewer.ID)
	require.Equal(t, int64(2), findReviewerInList("nobody@x.io", "u2", reviewers).Reviewer.ID)
	require.Nil(t, findReviewerInList("nobody@x.io", "zz", reviewers))
	require.Nil(t, findReviewerInList("a@b.c", "u", nil))
}

func TestSvbGetApplicableCodeOwners(t *testing.T) {
	ctx := context.Background()
	content := "# comment\n" +
		"* default@x.io\n" +
		"/docs/ docs@x.io\n" +
		"*.go go@x.io\n" +
		"/vendor/\n"

	tests := []struct {
		name  string
		files []string
		want  []string
	}{
		{"last matching rule wins", []string{"main.go"}, []string{"*.go"}},
		{"multiple files multiple rules", []string{"main.go", "README.md"}, []string{"*", "*.go"}},
		{"docs", []string{"docs/a.md"}, []string{"/docs/"}},
		{"ownership reset drops entry", []string{"vendor/x.go"}, []string{}},
		{"no files", nil, []string{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := &svbGit{content: content, files: tc.files}
			svc := svbService(g, nil, svbResolver{})
			got, err := svc.GetApplicableCodeOwners(ctx, svbRepo(), "main", "base", "head")
			require.NoError(t, err)
			require.Equal(t, svbFileSHA.String(), got.FileSHA)
			patterns := make([]string, 0, len(got.Entries))
			for _, e := range got.Entries {
				patterns = append(patterns, e.Pattern)
			}
			require.Equal(t, tc.want, patterns)
		})
	}

	t.Run("sorted by line number", func(t *testing.T) {
		g := &svbGit{content: content, files: []string{"main.go", "docs/a.md", "x.txt"}}
		got, err := svbService(g, nil, svbResolver{}).GetApplicableCodeOwners(ctx, svbRepo(), "main", "b", "h")
		require.NoError(t, err)
		require.Len(t, got.Entries, 3)
		require.Less(t, got.Entries[0].LineNumber, got.Entries[1].LineNumber)
		require.Less(t, got.Entries[1].LineNumber, got.Entries[2].LineNumber)
	})

	t.Run("file lookup error", func(t *testing.T) {
		g := &svbGit{treeErr: errSvbBoom}
		_, err := svbService(g, nil, svbResolver{}).GetApplicableCodeOwners(ctx, svbRepo(), "main", "b", "h")
		require.Error(t, err)
	})

	t.Run("diff error", func(t *testing.T) {
		g := &svbGit{content: content, diffErr: errSvbBoom}
		_, err := svbService(g, nil, svbResolver{}).GetApplicableCodeOwners(ctx, svbRepo(), "main", "b", "h")
		require.ErrorIs(t, err, errSvbBoom)
	})

	t.Run("pattern match error", func(t *testing.T) {
		g := &svbGit{content: "[ owner@x.io\n", files: []string{"a.txt"}}
		_, err := svbService(g, nil, svbResolver{}).GetApplicableCodeOwners(ctx, svbRepo(), "main", "b", "h")
		require.ErrorContains(t, err, "failed to match pattern")
	})
}

func TestSvbEvaluate(t *testing.T) {
	ctx := context.Background()
	alice := &types.Principal{ID: 1, UID: "alice", Email: "alice@x.io"}
	bob := &types.Principal{ID: 2, UID: "bob", Email: "bob@x.io"}
	author := &types.Principal{ID: 9, UID: "author", Email: "author@x.io"}
	ps := &svbPrincipalStore{
		byEmail: map[string]*types.Principal{
			"alice@x.io": alice, "bob@x.io": bob, "author@x.io": author,
		},
		byUID: map[string]*types.Principal{"alice": alice, "bob": bob},
	}
	resolver := svbResolver{groups: map[string]*types.UserGroup{
		"team": {Identifier: "team", Name: "The Team", Users: []string{"alice", "bob"}},
	}}
	pr := &types.PullReq{CreatedBy: 9, TargetBranch: "main", MergeBaseSHA: "b", SourceSHA: "h"}
	reviewers := []*types.PullReqReviewer{
		{
			Reviewer: *alice.ToPrincipalInfo(), ReviewDecision: enum.PullReqReviewDecisionApproved, SHA: "s1",
		},
	}

	t.Run("evaluates users and groups", func(t *testing.T) {
		g := &svbGit{
			content: "*.go alice@x.io bob@x.io author@x.io @team @missing ghost@x.io\n",
			files:   []string{"a.go"},
		}
		got, err := svbService(g, ps, resolver).Evaluate(ctx, svbRepo(), pr, reviewers)
		require.NoError(t, err)
		require.Equal(t, svbFileSHA.String(), got.FileSha)
		require.Len(t, got.EvaluationEntries, 1)

		entry := got.EvaluationEntries[0]
		require.Equal(t, "*.go", entry.Pattern)
		// The PR author and unknown users are skipped.
		require.Len(t, entry.UserEvaluations, 2)
		require.Equal(t, enum.PullReqReviewDecisionApproved, entry.UserEvaluations[0].ReviewDecision)
		require.Equal(t, "s1", entry.UserEvaluations[0].ReviewSHA)
		require.Equal(t, int64(2), entry.UserEvaluations[1].Owner.ID)
		require.Empty(t, entry.UserEvaluations[1].ReviewSHA)

		// Only the group member who reviewed is included.
		require.Len(t, entry.UserGroupEvaluations, 1)
		group := entry.UserGroupEvaluations[0]
		require.Equal(t, "team", group.Identifier)
		require.Equal(t, "The Team", group.Name)
		require.Len(t, group.Evaluations, 1)
		require.Equal(t, int64(1), group.Evaluations[0].Owner.ID)
	})

	t.Run("no applicable owners", func(t *testing.T) {
		g := &svbGit{content: "*.go alice@x.io\n", files: []string{"a.txt"}}
		got, err := svbService(g, ps, resolver).Evaluate(ctx, svbRepo(), pr, reviewers)
		require.NoError(t, err)
		require.Empty(t, got.EvaluationEntries)
	})

	t.Run("entry with only unresolvable owners is dropped", func(t *testing.T) {
		g := &svbGit{content: "*.go ghost@x.io\n", files: []string{"a.go"}}
		got, err := svbService(g, ps, resolver).Evaluate(ctx, svbRepo(), pr, reviewers)
		require.NoError(t, err)
		require.Empty(t, got.EvaluationEntries)
	})

	t.Run("lookup failures are skipped", func(t *testing.T) {
		g := &svbGit{content: "*.go alice@x.io @team\n", files: []string{"a.go"}}
		failing := &svbPrincipalStore{err: errSvbBoom}
		got, err := svbService(g, failing, svbResolver{err: errSvbBoom}).Evaluate(ctx, svbRepo(), pr, nil)
		require.NoError(t, err)
		require.Empty(t, got.EvaluationEntries)
	})

	t.Run("error loading owners", func(t *testing.T) {
		g := &svbGit{treeErr: errSvbBoom}
		got, err := svbService(g, ps, resolver).Evaluate(ctx, svbRepo(), pr, reviewers)
		require.Error(t, err)
		require.NotNil(t, got)
		require.Empty(t, got.EvaluationEntries)
	})
}

func TestSvbResolveUserGroupCodeOwner(t *testing.T) {
	ctx := context.Background()
	ps := &svbPrincipalStore{}
	svc := svbService(&svbGit{}, ps, svbResolver{groups: map[string]*types.UserGroup{
		"team": {Identifier: "team", Users: []string{"u"}},
	}})

	_, err := svc.resolveUserGroupCodeOwner(ctx, "other", nil)
	require.ErrorIs(t, err, usergroup.ErrNotFound)

	ps.err = errSvbBoom
	_, err = svc.resolveUserGroupCodeOwner(ctx, "team", nil)
	require.ErrorIs(t, err, errSvbBoom)

	ps.err = nil
	got, err := svc.resolveUserGroupCodeOwner(ctx, "team", nil)
	require.NoError(t, err)
	require.Empty(t, got.Evaluations)
}

func TestSvbResolveUserCodeOwnerByEmail(t *testing.T) {
	ctx := context.Background()
	alice := &types.Principal{ID: 1, UID: "alice", Email: "alice@x.io"}
	ps := &svbPrincipalStore{byEmail: map[string]*types.Principal{"alice@x.io": alice}}
	svc := svbService(&svbGit{}, ps, svbResolver{})

	got, err := svc.resolveUserCodeOwnerByEmail(ctx, "alice@x.io", nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), got.Owner.ID)
	require.Empty(t, got.ReviewDecision)

	_, err = svc.resolveUserCodeOwnerByEmail(ctx, "ghost@x.io", nil)
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
}

func TestSvbValidate(t *testing.T) {
	ctx := context.Background()
	alice := &types.Principal{ID: 1, UID: "alice", Email: "alice@x.io"}
	ps := &svbPrincipalStore{byEmail: map[string]*types.Principal{"alice@x.io": alice}}
	resolver := svbResolver{groups: map[string]*types.UserGroup{"team": {Identifier: "team"}}}

	t.Run("valid file", func(t *testing.T) {
		g := &svbGit{content: "* alice@x.io @team\n"}
		got, err := svbService(g, ps, resolver).Validate(ctx, svbRepo(), "")
		require.NoError(t, err)
		require.Empty(t, got.Violations)
	})

	t.Run("unknown owners are violations", func(t *testing.T) {
		g := &svbGit{content: "* ghost@x.io @nogroup\n"}
		got, err := svbService(g, ps, resolver).Validate(ctx, svbRepo(), "refs/heads/dev")
		require.NoError(t, err)
		require.Len(t, got.Violations, 2)
	})

	t.Run("file not found", func(t *testing.T) {
		g := &svbGit{treeErr: errSvbBoom}
		_, err := svbService(g, ps, resolver).Validate(ctx, svbRepo(), "")
		require.Error(t, err)
	})

	t.Run("too large", func(t *testing.T) {
		g := &svbGit{content: "* alice@x.io\n", size: maxGetContentFileSize + 1}
		got, err := svbService(g, ps, resolver).Validate(ctx, svbRepo(), "")
		require.NoError(t, err)
		require.Len(t, got.Violations, 1)
		require.Equal(t, enum.CodeOwnerViolationCodeFileTooLarge, got.Violations[0].Code)
	})

	t.Run("lookup outage is an error", func(t *testing.T) {
		g := &svbGit{content: "* alice@x.io\n"}
		_, err := svbService(g, &svbPrincipalStore{err: errSvbBoom}, resolver).Validate(ctx, svbRepo(), "")
		require.ErrorIs(t, err, errSvbBoom)

		g = &svbGit{content: "* @team\n"}
		_, err = svbService(g, ps, svbResolver{err: errSvbBoom}).Validate(ctx, svbRepo(), "")
		require.ErrorIs(t, err, errSvbBoom)
	})
}

func TestSvbValidateContent(t *testing.T) {
	ctx := context.Background()
	svc := svbService(&svbGit{}, &svbPrincipalStore{}, svbResolver{})

	got, err := svc.ValidateContent(ctx, []byte("pattern\\\n"))
	require.NoError(t, err)
	require.Len(t, got.Violations, 1)
	require.Equal(t, enum.CodeOwnerViolationCodeFileParseError, got.Violations[0].Code)

	got, err = svc.ValidateContent(ctx, []byte("[abc\n"))
	require.NoError(t, err)
	require.Len(t, got.Violations, 1)
	require.Equal(t, enum.CodeOwnerViolationCodePatternInvalid, got.Violations[0].Code)

	got, err = svc.ValidateContent(ctx, make([]byte, maxGetContentFileSize+1))
	require.NoError(t, err)
	require.Equal(t, enum.CodeOwnerViolationCodeFileTooLarge, got.Violations[0].Code)

	got, err = svc.ValidateContent(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, got.Violations)
}

func TestSvbProvideCodeOwners(t *testing.T) {
	require.NotNil(t, ProvideCodeOwners(&svbGit{}, nil, Config{}, &svbPrincipalStore{}, svbResolver{}))
}
