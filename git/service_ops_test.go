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

package git

import (
	"context"
	"io"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/git/sharedrepo"
	"github.com/harness/gitness/git/types"

	"github.com/stretchr/testify/require"
)

// noopHookClientFactory creates hook clients that accept every update.
type noopHookClientFactory struct{}

func (noopHookClientFactory) NewClient(map[string]string) (hook.Client, error) {
	return noopHookClient{}, nil
}

type noopHookClient struct{}

func (noopHookClient) PreReceive(context.Context, hook.PreReceiveInput) (hook.Output, error) {
	return hook.Output{}, nil
}

func (noopHookClient) Update(context.Context, hook.UpdateInput) (hook.Output, error) {
	return hook.Output{}, nil
}

func (noopHookClient) PostReceive(context.Context, hook.PostReceiveInput) (hook.Output, error) {
	return hook.Output{}, nil
}

// opsFixture is a service with one repository: main has a.txt, dir/b.txt and dir/sub/c.txt.
type opsFixture struct {
	svc      *Service
	repoUID  string
	repoPath string
	actor    Identity
}

func newOpsFixture(t *testing.T) *opsFixture {
	t.Helper()
	requireGit(t)

	config := types.Config{Root: t.TempDir(), HookPath: testGitHookPath}

	adapter, err := api.New(config, api.NewInMemoryLastCommitCache(time.Minute), nil)
	require.NoError(t, err)
	runnerFactory, err := sharedrepo.NewRunnerFactory(config.Root)
	require.NoError(t, err)
	svc, err := New(config, adapter, noopHookClientFactory{}, nil, runnerFactory)
	require.NoError(t, err)

	actor := Identity{Name: "Actor", Email: "actor@example.com"}
	out, err := svc.CreateRepository(t.Context(), &CreateRepositoryParams{
		Actor:         actor,
		DefaultBranch: "main",
		Files: []File{
			{Path: "a.txt", Content: []byte("a\n")},
			{Path: "dir/b.txt", Content: []byte("b\n")},
			{Path: "dir/sub/c.txt", Content: []byte("c\n")},
		},
	})
	require.NoError(t, err)

	return &opsFixture{
		svc:      svc,
		repoUID:  out.UID,
		repoPath: getFullPathForRepo(svc.reposRoot, out.UID),
		actor:    actor,
	}
}

func (f *opsFixture) read() ReadParams { return ReadParams{RepoUID: f.repoUID} }

func (f *opsFixture) write() WriteParams { return WriteParams{RepoUID: f.repoUID, Actor: f.actor} }

func (f *opsFixture) git(ctx context.Context, t *testing.T, args ...string) string {
	t.Helper()
	full := append([]string{
		"-c", "user.name=Fixture", "-c", "user.email=fixture@example.com",
		"-c", "maintenance.auto=false", "-c", "gc.auto=0",
	}, args...)

	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Dir = f.repoPath
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %s: %s", strings.Join(args, " "), out)
	return strings.TrimSpace(string(out))
}

func (f *opsFixture) head(ctx context.Context, t *testing.T) string {
	return f.git(ctx, t, "rev-parse", "main")
}

func TestService_BranchLifecycle(t *testing.T) {
	f := newOpsFixture(t)
	ctx := t.Context()
	head := f.head(ctx, t)

	t.Run("create", func(t *testing.T) {
		out, err := f.svc.CreateBranch(ctx, &CreateBranchParams{
			WriteParams: f.write(), BranchName: "feature", Target: "main",
		})
		require.NoError(t, err)
		require.Equal(t, "feature", out.Branch.Name)
		require.Equal(t, head, out.Branch.SHA.String())
		require.NotNil(t, out.Branch.Commit)
		require.Equal(t, head, f.git(ctx, t, "rev-parse", "refs/heads/feature"))
	})

	t.Run("create validation and conflicts", func(t *testing.T) {
		_, err := f.svc.CreateBranch(ctx, &CreateBranchParams{WriteParams: f.write(), BranchName: "feature", Target: "main"})
		require.True(t, errors.IsConflict(err), "got %v", err)

		_, err = f.svc.CreateBranch(ctx, &CreateBranchParams{WriteParams: f.write(), BranchName: "bad..name", Target: "main"})
		require.True(t, errors.IsInvalidArgument(err), "got %v", err)

		_, err = f.svc.CreateBranch(ctx, &CreateBranchParams{WriteParams: f.write(), BranchName: "x", Target: "missing"})
		require.Error(t, err)

		_, err = f.svc.CreateBranch(ctx, &CreateBranchParams{BranchName: "x", Target: "main"})
		require.Error(t, err)
	})

	t.Run("get", func(t *testing.T) {
		out, err := f.svc.GetBranch(ctx, &GetBranchParams{ReadParams: f.read(), BranchName: "refs/heads/feature"})
		require.NoError(t, err)
		require.Equal(t, "feature", out.Branch.Name)
		require.Equal(t, head, out.Branch.SHA.String())

		_, err = f.svc.GetBranch(ctx, &GetBranchParams{ReadParams: f.read(), BranchName: "missing"})
		require.Error(t, err)

		_, err = f.svc.GetBranch(ctx, nil)
		require.ErrorIs(t, err, ErrNoParamsProvided)
	})

	t.Run("list", func(t *testing.T) {
		out, err := f.svc.ListBranches(ctx, &ListBranchesParams{ReadParams: f.read(), IncludeCommit: true})
		require.NoError(t, err)
		require.Len(t, out.Branches, 2)
		require.True(t, out.IsLastPage)
		for _, b := range out.Branches {
			require.NotNil(t, b.Commit)
		}

		page, err := f.svc.ListBranches(ctx, &ListBranchesParams{
			ReadParams: f.read(), Page: 1, PageSize: 1, Sort: BranchSortOptionName,
		})
		require.NoError(t, err)
		require.Len(t, page.Branches, 1)
		require.False(t, page.IsLastPage)
		require.Equal(t, "feature", page.Branches[0].Name)

		filtered, err := f.svc.ListBranches(ctx, &ListBranchesParams{
			ReadParams: f.read(), Query: "feat", Sort: BranchSortOptionDate, Order: SortOrderDesc,
		})
		require.NoError(t, err)
		require.Len(t, filtered.Branches, 1)

		_, err = f.svc.ListBranches(ctx, nil)
		require.ErrorIs(t, err, ErrNoParamsProvided)
	})

	t.Run("delete", func(t *testing.T) {
		require.NoError(t, f.svc.DeleteBranch(ctx, &DeleteBranchParams{
			WriteParams: f.write(), BranchName: "feature", SHA: head,
		}))
		exists, err := f.svc.git.IsBranchExist(ctx, f.repoPath, "feature")
		require.NoError(t, err)
		require.False(t, exists)

		require.ErrorIs(t, f.svc.DeleteBranch(ctx, nil), ErrNoParamsProvided)
	})
}

func TestService_TagLifecycle(t *testing.T) {
	f := newOpsFixture(t)
	ctx := t.Context()
	head := f.head(ctx, t)
	when := time.Date(2024, 3, 4, 5, 6, 7, 0, time.UTC)

	t.Run("validation", func(t *testing.T) {
		_, err := f.svc.CreateCommitTag(ctx, nil)
		require.ErrorIs(t, err, ErrNoParamsProvided)
		_, err = f.svc.CreateCommitTag(ctx, &CreateCommitTagParams{WriteParams: f.write(), Target: "main"})
		require.Error(t, err)
		_, err = f.svc.CreateCommitTag(ctx, &CreateCommitTagParams{WriteParams: f.write(), Name: "v"})
		require.Error(t, err)
		_, err = f.svc.CreateCommitTag(ctx, &CreateCommitTagParams{WriteParams: f.write(), Name: "v", Target: "missing"})
		require.True(t, errors.IsNotFound(err), "got %v", err)
	})

	// NOTE: lightweight tags are created with the git CLI, the service always reads back an annotated tag.
	f.git(ctx, t, "tag", "light", head)

	t.Run("annotated with tagger override", func(t *testing.T) {
		tagger := Identity{Name: "Tagger", Email: "tagger@example.com"}
		out, err := f.svc.CreateCommitTag(ctx, &CreateCommitTagParams{
			WriteParams: f.write(), Name: "annotated", Target: "main", Message: "release notes",
			Tagger: &tagger, TaggerDate: &when,
		})
		require.NoError(t, err)
		require.True(t, out.IsAnnotated)
		require.Equal(t, "release notes", out.Title)
		require.Equal(t, "Tagger", out.Tagger.Identity.Name)
		require.Equal(t, "annotated", f.git(ctx, t, "tag", "-l", "annotated"))
	})

	t.Run("duplicate", func(t *testing.T) {
		_, err := f.svc.CreateCommitTag(ctx, &CreateCommitTagParams{
			WriteParams: f.write(), Name: "light", Target: "main", Message: "again",
		})
		require.True(t, errors.IsConflict(err), "got %v", err)
	})

	t.Run("list", func(t *testing.T) {
		// tags that do not point to commits are filtered out
		f.git(ctx, t, "tag", "blob-tag", f.git(ctx, t, "rev-parse", "main:a.txt"))

		out, err := f.svc.ListCommitTags(ctx, &ListCommitTagsParams{ReadParams: f.read(), IncludeCommit: true})
		require.NoError(t, err)
		names := map[string]CommitTag{}
		for _, tag := range out.Tags {
			names[tag.Name] = tag
		}
		require.Len(t, names, 2)
		require.False(t, names["light"].IsAnnotated)
		require.True(t, names["annotated"].IsAnnotated)
		require.Equal(t, "release notes", strings.TrimSpace(names["annotated"].Message))
		require.NotNil(t, names["annotated"].Commit)
		require.NotNil(t, names["light"].Commit)

		page, err := f.svc.ListCommitTags(ctx, &ListCommitTagsParams{
			ReadParams: f.read(), Query: "light", Sort: TagSortOptionName, Order: SortOrderAsc, Page: 1, PageSize: 10,
		})
		require.NoError(t, err)
		require.Len(t, page.Tags, 1)

		_, err = f.svc.ListCommitTags(ctx, &ListCommitTagsParams{ReadParams: ReadParams{}})
		require.Error(t, err)
	})

	t.Run("delete", func(t *testing.T) {
		require.NoError(t, f.svc.DeleteTag(ctx, &DeleteTagParams{WriteParams: f.write(), Name: "light"}))
		require.Empty(t, f.git(ctx, t, "tag", "-l", "light"))

		require.Error(t, f.svc.DeleteTag(ctx, &DeleteTagParams{WriteParams: f.write()}))
	})
}

func TestService_Refs(t *testing.T) {
	f := newOpsFixture(t)
	ctx := t.Context()
	head := f.head(ctx, t)
	headSHA := sha.Must(head)

	t.Run("get ref", func(t *testing.T) {
		out, err := f.svc.GetRef(ctx, GetRefParams{ReadParams: f.read(), Name: "main", Type: enum.RefTypeBranch})
		require.NoError(t, err)
		require.Equal(t, head, out.SHA.String())

		_, err = f.svc.GetRef(ctx, GetRefParams{ReadParams: f.read(), Name: "missing", Type: enum.RefTypeBranch})
		require.Error(t, err)

		_, err = f.svc.GetRef(ctx, GetRefParams{ReadParams: f.read(), Type: enum.RefTypeBranch})
		require.Error(t, err)

		var nilParams *GetRefParams
		require.ErrorIs(t, nilParams.Validate(), ErrNoParamsProvided)
	})

	t.Run("update ref", func(t *testing.T) {
		require.NoError(t, f.svc.UpdateRef(ctx, UpdateRefParams{
			WriteParams: f.write(), Type: enum.RefTypeBranch, Name: "created", NewValue: headSHA, OldValue: sha.Nil,
		}))
		require.Equal(t, head, f.git(ctx, t, "rev-parse", "refs/heads/created"))

		require.NoError(t, f.svc.UpdateRef(ctx, UpdateRefParams{
			WriteParams: f.write(), Type: enum.RefTypeBranch, Name: "created", NewValue: sha.Nil, OldValue: headSHA,
		}))
		exists, err := f.svc.git.IsBranchExist(ctx, f.repoPath, "created")
		require.NoError(t, err)
		require.False(t, exists)

		require.Error(t, f.svc.UpdateRef(ctx, UpdateRefParams{Type: enum.RefTypeBranch, Name: "x"}))
	})

	t.Run("update refs", func(t *testing.T) {
		require.NoError(t, f.svc.UpdateRefs(ctx, UpdateRefsParams{
			WriteParams: f.write(),
			Refs: []RefUpdate{
				{Name: "refs/heads/multi1", Old: sha.Nil, New: headSHA},
				{Name: "refs/heads/multi2", Old: sha.Nil, New: headSHA},
			},
		}))
		require.Equal(t, head, f.git(ctx, t, "rev-parse", "refs/heads/multi1"))
		require.Equal(t, head, f.git(ctx, t, "rev-parse", "refs/heads/multi2"))

		require.Error(t, f.svc.UpdateRefs(ctx, UpdateRefsParams{}))
	})
}

func TestService_TreeAndPaths(t *testing.T) {
	f := newOpsFixture(t)
	ctx := t.Context()

	t.Run("get tree node", func(t *testing.T) {
		out, err := f.svc.GetTreeNode(ctx, &GetTreeNodeParams{
			ReadParams: f.read(), GitREF: "main", Path: "dir/b.txt", IncludeLatestCommit: true,
		})
		require.NoError(t, err)
		require.Equal(t, TreeNodeTypeBlob, out.Node.Type)
		require.Equal(t, TreeNodeModeFile, out.Node.Mode)
		require.Equal(t, "b.txt", out.Node.Name)
		require.NotNil(t, out.Commit)
		require.Equal(t, f.head(ctx, t), out.Commit.SHA.String())

		out, err = f.svc.GetTreeNode(ctx, &GetTreeNodeParams{ReadParams: f.read(), GitREF: "main", Path: "dir"})
		require.NoError(t, err)
		require.Equal(t, TreeNodeTypeTree, out.Node.Type)
		require.Nil(t, out.Commit)

		_, err = f.svc.GetTreeNode(ctx, &GetTreeNodeParams{ReadParams: f.read(), GitREF: "main", Path: "nope"})
		require.Error(t, err)
	})

	t.Run("list tree nodes", func(t *testing.T) {
		out, err := f.svc.ListTreeNodes(ctx, &ListTreeNodeParams{ReadParams: f.read(), GitREF: "main", Path: ""})
		require.NoError(t, err)
		var names []string
		for _, n := range out.Nodes {
			names = append(names, n.Name)
		}
		require.ElementsMatch(t, []string{"a.txt", "dir"}, names)

		out, err = f.svc.ListTreeNodes(ctx, &ListTreeNodeParams{
			ReadParams: f.read(), GitREF: "main", Path: "dir", FlattenDirectories: true,
		})
		require.NoError(t, err)
		require.Len(t, out.Nodes, 2)

		_, err = f.svc.ListTreeNodes(ctx, &ListTreeNodeParams{ReadParams: f.read(), GitREF: "main", Path: "nope"})
		require.Error(t, err)
	})

	t.Run("list paths", func(t *testing.T) {
		out, err := f.svc.ListPaths(ctx, &ListPathsParams{ReadParams: f.read(), GitREF: "main", IncludeDirectories: true})
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"a.txt", "dir/b.txt", "dir/sub/c.txt"}, out.Files)
		require.ElementsMatch(t, []string{"dir", "dir/sub"}, out.Directories)

		_, err = f.svc.ListPaths(ctx, &ListPathsParams{ReadParams: f.read(), GitREF: "missing"})
		require.Error(t, err)
	})

	t.Run("paths details", func(t *testing.T) {
		out, err := f.svc.PathsDetails(ctx, PathsDetailsParams{
			ReadParams: f.read(), GitREF: "main", Paths: []string{"a.txt", "dir"},
		})
		require.NoError(t, err)
		require.Len(t, out.Details, 2)
		require.Equal(t, "a.txt", out.Details[0].Path)
		require.NotNil(t, out.Details[0].LastCommit)

		_, err = f.svc.PathsDetails(ctx, PathsDetailsParams{
			ReadParams: f.read(), GitREF: "missing", Paths: []string{"a.txt"},
		})
		require.Error(t, err)
		_, err = f.svc.PathsDetails(ctx, PathsDetailsParams{})
		require.Error(t, err)
	})
}

func TestService_Blob(t *testing.T) {
	f := newOpsFixture(t)
	ctx := t.Context()
	blobSHA := f.git(ctx, t, "rev-parse", "main:a.txt")

	out, err := f.svc.GetBlob(ctx, &GetBlobParams{ReadParams: f.read(), SHA: blobSHA})
	require.NoError(t, err)
	defer out.Content.Close()
	require.Equal(t, blobSHA, out.SHA.String())
	require.Equal(t, int64(2), out.Size)
	data, err := io.ReadAll(out.Content)
	require.NoError(t, err)
	require.Equal(t, "a\n", string(data))

	limited, err := f.svc.GetBlob(ctx, &GetBlobParams{ReadParams: f.read(), SHA: blobSHA, SizeLimit: 1})
	require.NoError(t, err)
	defer limited.Content.Close()
	require.Equal(t, int64(1), limited.ContentSize)

	_, err = f.svc.GetBlob(ctx, nil)
	require.ErrorIs(t, err, ErrNoParamsProvided)
	_, err = f.svc.GetBlob(ctx, &GetBlobParams{ReadParams: f.read(), SHA: strings.Repeat("1", 40)})
	require.Error(t, err)
}

func TestService_Commits(t *testing.T) {
	f := newOpsFixture(t)
	ctx := t.Context()
	first := f.head(ctx, t)

	tree := f.git(ctx, t, "rev-parse", "main^{tree}")
	one := f.git(ctx, t, "commit-tree", tree, "-p", "main", "-m", "feature one")
	two := f.git(ctx, t, "commit-tree", tree, "-p", one, "-m", "feature two")
	f.git(ctx, t, "update-ref", "refs/heads/feature", two)

	t.Run("get commit", func(t *testing.T) {
		out, err := f.svc.GetCommit(ctx, &GetCommitParams{ReadParams: f.read(), Revision: "feature"})
		require.NoError(t, err)
		require.Equal(t, "feature two", out.Commit.Title)
		require.Len(t, out.Commit.ParentSHAs, 1)

		_, err = f.svc.GetCommit(ctx, &GetCommitParams{ReadParams: f.read(), Revision: "missing"})
		require.Error(t, err)
		_, err = f.svc.GetCommit(ctx, nil)
		require.ErrorIs(t, err, ErrNoParamsProvided)
	})

	t.Run("list commits", func(t *testing.T) {
		out, err := f.svc.ListCommits(ctx, &ListCommitsParams{
			ReadParams: f.read(), GitREF: "feature", Page: 1, Limit: 10,
		})
		require.NoError(t, err)
		require.Len(t, out.Commits, 3)
		require.Equal(t, 3, out.TotalCommits)

		// total commits is computed through the divergence against After
		paged, err := f.svc.ListCommits(ctx, &ListCommitsParams{
			ReadParams: f.read(), GitREF: "feature", After: "main", Page: 1, Limit: 1,
		})
		require.NoError(t, err)
		require.Len(t, paged.Commits, 1)
		require.Equal(t, 2, paged.TotalCommits)

		stats, err := f.svc.ListCommits(ctx, &ListCommitsParams{
			ReadParams: f.read(), GitREF: "main", Page: 1, Limit: 10, IncludeStats: true, Path: "a.txt",
		})
		require.NoError(t, err)
		require.Len(t, stats.Commits, 1)
		require.Equal(t, first, stats.Commits[0].SHA.String())

		_, err = f.svc.ListCommits(ctx, nil)
		require.ErrorIs(t, err, ErrNoParamsProvided)
	})

	t.Run("divergences", func(t *testing.T) {
		out, err := f.svc.GetCommitDivergences(ctx, &GetCommitDivergencesParams{
			ReadParams: f.read(),
			Requests: []CommitDivergenceRequest{
				{From: "feature", To: "main"},
				{From: "main", To: "missing"},
			},
		})
		require.NoError(t, err)
		require.Equal(t, []api.CommitDivergence{{Ahead: 2, Behind: 0}, {Ahead: -1, Behind: -1}}, out.Divergences)

		_, err = f.svc.GetCommitDivergences(ctx, nil)
		require.ErrorIs(t, err, ErrNoParamsProvided)
	})
}

func TestCommitMessage(t *testing.T) {
	require.Equal(t, "subject", CommitMessage("subject", ""))
	require.Equal(t, "subject\n\nbody", CommitMessage("subject", "body"))
}

func TestIdentityAndSignatureStrings(t *testing.T) {
	id := Identity{Name: "Jane", Email: "jane@example.com"}
	require.Equal(t, "Jane <jane@example.com>", id.String())
	require.NoError(t, id.Validate())
	require.Error(t, (&Identity{Email: "x"}).Validate())
	require.Error(t, (&Identity{Name: "x"}).Validate())

	when := time.Unix(0, 0).UTC()
	sig := Signature{Identity: id, When: when}
	require.True(t, strings.HasPrefix(sig.String(), "Jane <jane@example.com> "))
}
