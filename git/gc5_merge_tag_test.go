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
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"

	"github.com/stretchr/testify/require"
)

// gc5Diverged prepares a repository where main and feature both moved on from the same base commit.
// feature adds feat.txt, main adds other.txt.
type gc5Diverged struct {
	*gc5Fixture
	base, mainTip, featureTip sha.SHA
}

func gc5NewDiverged(t *testing.T) *gc5Diverged {
	t.Helper()

	f := gc5NewFixture(t)
	ctx := t.Context()
	base := f.head(ctx, t)

	_, err := f.svc.CommitFiles(ctx, &CommitFilesParams{
		WriteParams: f.write(), Branch: "main", NewBranch: "feature", Message: "feature work",
		Actions: []CommitFileAction{{Action: CreateAction, Path: "feat.txt", Payload: []byte("feat\n")}},
	})
	require.NoError(t, err)
	featureTip := sha.Must(f.git(ctx, t, "rev-parse", "refs/heads/feature"))

	f.commit(ctx, t, CommitFileAction{Action: CreateAction, Path: "other.txt", Payload: []byte("other\n")})

	return &gc5Diverged{gc5Fixture: f, base: base, mainTip: f.head(ctx, t), featureTip: featureTip}
}

func TestGc5MergeMethods(t *testing.T) {
	d := gc5NewDiverged(t)
	ctx := t.Context()

	author := Identity{Name: "Gc5Author", Email: "gc5-author@example.com"}
	when := time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)

	methods := []enum.MergeMethod{enum.MergeMethodMerge, enum.MergeMethodSquash, enum.MergeMethodRebase, ""}
	for _, method := range methods {
		t.Run("method "+string(method), func(t *testing.T) {
			ref := "refs/heads/merged-" + string(method) + "x"

			out, err := d.svc.Merge(ctx, &MergeParams{
				WriteParams: d.write(),
				BaseBranch:  "main",
				HeadBranch:  "feature",
				Message:     "gc5 merge",
				Method:      method,
				Author:      &author, AuthorDate: &when, Committer: &author, CommitterDate: &when,
				HeadBranchExpectedSHA: d.featureTip,
				Refs:                  []RefUpdate{{Name: ref, Old: sha.Nil}},
			})
			require.NoError(t, err)
			require.Equal(t, d.mainTip, out.BaseSHA)
			require.Equal(t, d.featureTip, out.HeadSHA)
			require.Equal(t, d.base, out.MergeBaseSHA)
			require.False(t, out.MergeSHA.IsEmpty())
			require.Equal(t, 1, out.CommitCount)
			require.Equal(t, 1, out.ChangedFileCount)
			require.Equal(t, 1, out.Additions)
			require.Empty(t, out.ConflictFiles)

			require.Equal(t, out.MergeSHA.String(), d.git(ctx, t, "rev-parse", ref))
			if method != enum.MergeMethodRebase {
				// a rebase keeps the authors of the replayed commits.
				require.Equal(t, "Gc5Author", d.git(ctx, t, "log", "-1", "--format=%an", ref))
			}
			// both files end up on the merged branch.
			files := d.git(ctx, t, "ls-tree", "-r", "--name-only", ref)
			require.Contains(t, files, "feat.txt")
			require.Contains(t, files, "other.txt")
		})
	}
}

func TestGc5MergeFastForward(t *testing.T) {
	f := gc5NewFixture(t)
	ctx := t.Context()
	base := f.head(ctx, t)

	_, err := f.svc.CommitFiles(ctx, &CommitFilesParams{
		WriteParams: f.write(), Branch: "main", NewBranch: "feature", Message: "feature work",
		Actions: []CommitFileAction{{Action: CreateAction, Path: "feat.txt", Payload: []byte("feat\n")}},
	})
	require.NoError(t, err)
	featureTip := sha.Must(f.git(ctx, t, "rev-parse", "refs/heads/feature"))

	out, err := f.svc.Merge(ctx, &MergeParams{
		WriteParams: f.write(), BaseBranch: "main", HeadBranch: "feature", Method: enum.MergeMethodFastForward,
		Refs: []RefUpdate{{Name: "refs/heads/main", Old: base}},
	})
	require.NoError(t, err)
	require.Equal(t, featureTip, out.MergeSHA)
	require.Equal(t, featureTip, f.head(ctx, t))
}

func TestGc5MergeConflicts(t *testing.T) {
	f := gc5NewFixture(t)
	ctx := t.Context()
	aSHA := sha.Must(f.git(ctx, t, "rev-parse", "main:a.txt"))

	_, err := f.svc.CommitFiles(ctx, &CommitFilesParams{
		WriteParams: f.write(), Branch: "main", NewBranch: "feature", Message: "feature edit",
		Actions: []CommitFileAction{{Action: UpdateAction, Path: "a.txt", SHA: aSHA, Payload: []byte("feature\n")}},
	})
	require.NoError(t, err)
	f.commit(ctx, t, CommitFileAction{Action: UpdateAction, Path: "a.txt", SHA: aSHA, Payload: []byte("main\n")})

	out, err := f.svc.Merge(ctx, &MergeParams{
		WriteParams: f.write(), BaseBranch: "main", HeadBranch: "feature", Method: enum.MergeMethodMerge,
		Refs: []RefUpdate{{Name: "refs/heads/merged", Old: sha.Nil}},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"a.txt"}, out.ConflictFiles)
	require.True(t, out.MergeSHA.IsNil() || out.MergeSHA.IsEmpty())
	require.Equal(t, 1, out.CommitCount)

	// the conflicting merge must not have created the reference.
	branches, err := f.svc.ListBranches(ctx, &ListBranchesParams{ReadParams: f.read()})
	require.NoError(t, err)
	for _, b := range branches.Branches {
		require.NotEqual(t, "merged", b.Name)
	}

	// the dry run reports the same conflicts and persists nothing.
	head := sha.Must(f.git(ctx, t, "rev-parse", "refs/heads/feature"))
	verify, err := f.svc.MergeVerify(ctx, &MergeVerifyParams{
		ReadParams: f.read(), BaseSHA: f.head(ctx, t), HeadSHA: head, Method: enum.MergeMethodMerge,
	})
	require.NoError(t, err)
	require.Equal(t, []string{"a.txt"}, verify.ConflictFiles)
}

func TestGc5MergeErrors(t *testing.T) {
	d := gc5NewDiverged(t)
	ctx := t.Context()

	valid := MergeParams{WriteParams: d.write(), BaseBranch: "main", HeadBranch: "feature", Message: "m"}

	tests := []struct {
		name   string
		mutate func(p *MergeParams)
		check  func(t *testing.T, err error)
	}{
		{"invalid params", func(p *MergeParams) { p.RepoUID = "" }, nil},
		{"unsupported method", func(p *MergeParams) { p.Method = "bogus" }, func(t *testing.T, err error) {
			require.True(t, errors.IsInvalidArgument(err), "got %v", err)
		}},
		{"unknown base branch", func(p *MergeParams) { p.BaseBranch = "nope" }, nil},
		{"unknown head branch", func(p *MergeParams) { p.HeadBranch = "nope" }, nil},
		{"head moved", func(p *MergeParams) {
			p.HeadBranchExpectedSHA = sha.Must(strings.Repeat("2", 40))
		}, func(t *testing.T, err error) {
			require.True(t, errors.IsPreconditionFailed(err), "got %v", err)
		}},
		{"head without new commits", func(p *MergeParams) {
			p.HeadBranch = ""
			p.HeadSHA = d.base
		}, func(t *testing.T, err error) {
			require.True(t, errors.IsInvalidArgument(err), "got %v", err)
		}},
		{"stale reference update", func(p *MergeParams) {
			p.Refs = []RefUpdate{{Name: "refs/heads/main", Old: sha.Must(strings.Repeat("3", 40))}}
		}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := valid
			tt.mutate(&p)

			_, err := d.svc.Merge(ctx, &p)
			require.Error(t, err)
			if tt.check != nil {
				tt.check(t, err)
			}
		})
	}
}

func TestGc5MergeVerify(t *testing.T) {
	d := gc5NewDiverged(t)
	ctx := t.Context()

	for _, method := range []enum.MergeMethod{
		enum.MergeMethodMerge, enum.MergeMethodSquash, enum.MergeMethodRebase, "",
	} {
		t.Run("method "+string(method), func(t *testing.T) {
			out, err := d.svc.MergeVerify(ctx, &MergeVerifyParams{
				ReadParams: d.read(), BaseSHA: d.mainTip, HeadSHA: d.featureTip, Method: method,
			})
			require.NoError(t, err)
			require.Equal(t, d.base, out.MergeBaseSHA)
			require.Equal(t, 1, out.CommitCount)
			require.Equal(t, 1, out.ChangedFileCount)
			require.Empty(t, out.ConflictFiles)
		})
	}

	// nothing is persisted: the objects of the dry run are discarded and no refs changed.
	require.Equal(t, d.mainTip, d.head(ctx, t))

	tests := []struct {
		name   string
		params MergeVerifyParams
	}{
		{"invalid params", MergeVerifyParams{}},
		{"unsupported method", MergeVerifyParams{
			ReadParams: d.read(), BaseSHA: d.mainTip, HeadSHA: d.featureTip, Method: "bogus",
		}},
		{"head without new commits", MergeVerifyParams{ReadParams: d.read(), BaseSHA: d.mainTip, HeadSHA: d.base}},
		{"unknown commits", MergeVerifyParams{
			ReadParams: d.read(), BaseSHA: sha.Must(strings.Repeat("4", 40)), HeadSHA: d.featureTip,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := d.svc.MergeVerify(ctx, &tt.params)
			require.Error(t, err)
		})
	}
}

func TestGc5MergeBaseAndAncestor(t *testing.T) {
	d := gc5NewDiverged(t)
	ctx := t.Context()

	out, err := d.svc.MergeBase(ctx, MergeBaseParams{ReadParams: d.read(), Ref1: "main", Ref2: "feature"})
	require.NoError(t, err)
	require.Equal(t, d.base, out.MergeBaseSHA)

	_, err = d.svc.MergeBase(ctx, MergeBaseParams{ReadParams: d.read(), Ref1: "main", Ref2: "nope"})
	require.Error(t, err)

	anc, err := d.svc.IsAncestor(ctx, IsAncestorParams{
		ReadParams: d.read(), AncestorCommitSHA: d.base, DescendantCommitSHA: d.featureTip,
	})
	require.NoError(t, err)
	require.True(t, anc.Ancestor)

	anc, err = d.svc.IsAncestor(ctx, IsAncestorParams{
		ReadParams: d.read(), AncestorCommitSHA: d.featureTip, DescendantCommitSHA: d.mainTip,
	})
	require.NoError(t, err)
	require.False(t, anc.Ancestor)

	_, err = d.svc.IsAncestor(ctx, IsAncestorParams{
		ReadParams: ReadParams{RepoUID: "does-not-exist"}, AncestorCommitSHA: d.base, DescendantCommitSHA: d.mainTip,
	})
	require.Error(t, err)
}

func TestGc5CommitTags(t *testing.T) {
	f := gc5NewFixture(t)
	ctx := t.Context()
	tagger := Identity{Name: "Gc5Tagger", Email: "gc5-tagger@example.com"}
	when := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

	for _, p := range []*CreateCommitTagParams{
		nil,
		{WriteParams: f.write(), Target: "main"},
		{WriteParams: f.write(), Name: "v0"},
	} {
		_, err := f.svc.CreateCommitTag(ctx, p)
		require.Error(t, err)
	}

	// NOTE: creating a lightweight tag through the service fails today (see PR observations), so use git for it.
	f.git(ctx, t, "tag", "light", "main")

	annotated, err := f.svc.CreateCommitTag(ctx, &CreateCommitTagParams{
		WriteParams: f.write(), Name: "annotated", Target: "main", Message: "release one\n\ndetails",
		Tagger: &tagger, TaggerDate: &when,
	})
	require.NoError(t, err)
	require.True(t, annotated.IsAnnotated)
	require.Equal(t, "release one", annotated.Title)
	require.Equal(t, "Gc5Tagger", annotated.Tagger.Identity.Name)
	require.Equal(t, f.head(ctx, t), annotated.Commit.SHA)

	_, err = f.svc.CreateCommitTag(ctx, &CreateCommitTagParams{WriteParams: f.write(), Name: "light", Target: "main"})
	require.True(t, errors.IsConflict(err), "got %v", err)

	_, err = f.svc.CreateCommitTag(ctx, &CreateCommitTagParams{WriteParams: f.write(), Name: "other", Target: "nope"})
	require.True(t, errors.IsNotFound(err), "got %v", err)

	list, err := f.svc.ListCommitTags(ctx, &ListCommitTagsParams{ReadParams: f.read(), IncludeCommit: true})
	require.NoError(t, err)
	names := map[string]bool{}
	for _, tag := range list.Tags {
		names[tag.Name] = tag.IsAnnotated
	}
	require.Equal(t, map[string]bool{"light": false, "annotated": true}, names)

	require.Error(t, f.svc.DeleteTag(ctx, &DeleteTagParams{WriteParams: f.write()}))
	require.NoError(t, f.svc.DeleteTag(ctx, &DeleteTagParams{WriteParams: f.write(), Name: "light"}))
	// deleting a tag that no longer exists is not an error.
	require.NoError(t, f.svc.DeleteTag(ctx, &DeleteTagParams{WriteParams: f.write(), Name: "light"}))
}
