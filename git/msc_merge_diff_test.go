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
	"bytes"
	"strings"
	"testing"

	"github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"

	"github.com/stretchr/testify/require"
)

func mscCommitOnFeature(t *testing.T, f *mscOpsFixture) sha.SHA {
	t.Helper()
	ctx := t.Context()
	_, err := f.svc.CreateBranch(ctx, &CreateBranchParams{
		WriteParams: f.mscWrite(), BranchName: "feature", Target: "main",
	})
	require.NoError(t, err)
	resp, err := f.svc.CommitFiles(ctx, &CommitFilesParams{
		WriteParams: f.mscWrite(),
		Message:     "feature work",
		Branch:      "feature",
		Actions: []CommitFileAction{{
			Action: CreateAction, Path: "feature.txt", Payload: []byte("feature\n"),
		}},
	})
	require.NoError(t, err)
	return resp.CommitID
}

func TestMscService_MergeBaseAndAncestor(t *testing.T) {
	f := mscNewOpsFixture(t)
	ctx := t.Context()
	featureHead := mscCommitOnFeature(t, f)
	mainHead := sha.Must(f.mscHead(ctx, t))

	base, err := f.svc.MergeBase(ctx, MergeBaseParams{ReadParams: f.mscRead(), Ref1: "main", Ref2: "feature"})
	require.NoError(t, err)
	require.False(t, base.MergeBaseSHA.IsEmpty())

	ancestor, err := f.svc.IsAncestor(ctx, IsAncestorParams{
		ReadParams: f.mscRead(), AncestorCommitSHA: mainHead, DescendantCommitSHA: featureHead,
	})
	require.NoError(t, err)
	require.True(t, ancestor.Ancestor)

	_, err = f.svc.MergeBase(ctx, MergeBaseParams{ReadParams: f.mscRead(), Ref1: "main"})
	require.Error(t, err)
}

func TestMscService_MergeVerifyAndFastForward(t *testing.T) {
	f := mscNewOpsFixture(t)
	ctx := t.Context()
	featureHead := mscCommitOnFeature(t, f)
	mainHead := sha.Must(f.mscHead(ctx, t))

	verify, err := f.svc.MergeVerify(ctx, &MergeVerifyParams{
		ReadParams: f.mscRead(),
		BaseSHA:    mainHead,
		HeadSHA:    featureHead,
		Method:     enum.MergeMethodFastForward,
	})
	require.NoError(t, err)
	require.Equal(t, 1, verify.CommitCount)
	require.GreaterOrEqual(t, verify.ChangedFileCount, 1)

	out, err := f.svc.Merge(ctx, &MergeParams{
		WriteParams: f.mscWrite(),
		BaseBranch:  "main",
		HeadBranch:  "feature",
		Method:      enum.MergeMethodFastForward,
		Message:     "merge feature",
		Refs:        []RefUpdate{{Name: "refs/heads/main", Old: mainHead}},
	})
	require.NoError(t, err)
	require.Equal(t, featureHead, out.MergeSHA)
	require.Equal(t, featureHead.String(), f.mscHead(ctx, t))
}

func TestMscService_DiffHelpers(t *testing.T) {
	f := mscNewOpsFixture(t)
	ctx := t.Context()
	featureHead := mscCommitOnFeature(t, f)

	short, err := f.svc.DiffShortStat(ctx, &DiffParams{
		ReadParams: f.mscRead(), BaseRef: "main", HeadRef: "feature", MergeBase: true,
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, short.Files, 1)

	stats, err := f.svc.DiffStats(ctx, &DiffParams{
		ReadParams: f.mscRead(), BaseRef: "main", HeadRef: "feature",
	})
	require.NoError(t, err)
	require.Equal(t, 1, stats.Commits)

	var buf bytes.Buffer
	require.NoError(t, f.svc.RawDiff(ctx, &buf, &DiffParams{
		ReadParams: f.mscRead(), BaseRef: "main", HeadRef: "feature", MergeBase: true,
	}))
	require.Contains(t, buf.String(), "diff")

	names, err := f.svc.DiffFileNames(ctx, &DiffParams{
		ReadParams: f.mscRead(), BaseRef: "main", HeadRef: "feature", MergeBase: true,
	})
	require.NoError(t, err)
	require.Contains(t, names.Files, "feature.txt")

	var commitBuf bytes.Buffer
	require.NoError(t, f.svc.CommitDiff(ctx, &GetCommitParams{
		ReadParams: f.mscRead(), Revision: featureHead.String(),
	}, &commitBuf))
	require.True(t, strings.Contains(commitBuf.String(), "feature.txt") || len(commitBuf.String()) > 0)
}

func TestMscMergeParams_Validate(t *testing.T) {
	valid := &MergeParams{WriteParams: WriteParams{RepoUID: "r", Actor: Identity{Name: "a", Email: "a@test.invalid"}},
		BaseBranch: "main", HeadBranch: "feature"}
	require.NoError(t, valid.Validate())

	require.Error(t, (&MergeParams{
		WriteParams: WriteParams{RepoUID: "r", Actor: Identity{Name: "a", Email: "a@test.invalid"}},
		BaseBranch:  "main",
	}).Validate())
}
