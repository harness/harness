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

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/diff"
	"github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	tenum "github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestGc5CommitFilesActions(t *testing.T) {
	f := gc5NewFixture(t)
	ctx := t.Context()

	aSHA := sha.Must(f.git(ctx, t, "rev-parse", "main:a.txt"))
	bSHA := sha.Must(f.git(ctx, t, "rev-parse", "main:dir/b.txt"))

	out := f.commit(ctx, t,
		CommitFileAction{Action: CreateAction, Path: "new.txt", Payload: []byte("new\n")},
		CommitFileAction{Action: UpdateAction, Path: "a.txt", SHA: aSHA, Payload: []byte("a2\n")},
		CommitFileAction{Action: DeleteAction, Path: "dir/sub/c.go"},
		CommitFileAction{Action: MoveAction, Path: "dir/b.txt", SHA: bSHA, Payload: []byte("moved/b.txt")},
	)
	require.Len(t, out.ChangedFiles, 4)
	require.Equal(t, "new\n", f.git(ctx, t, "show", "main:new.txt")+"\n")
	require.Equal(t, "a2", f.git(ctx, t, "show", "main:a.txt"))
	require.Empty(t, f.git(ctx, t, "ls-tree", "-r", "--name-only", "main", "dir/sub/c.go"))
	require.Equal(t, "b", f.git(ctx, t, "show", "main:moved/b.txt"))
}

func TestGc5CommitFilesPatchText(t *testing.T) {
	f := gc5NewFixture(t)
	ctx := t.Context()

	f.commit(ctx, t, CommitFileAction{Action: CreateAction, Path: "lines.txt", Payload: []byte("1\n2\n3\n")})
	fileSHA := sha.Must(f.git(ctx, t, "rev-parse", "main:lines.txt"))

	out := f.commit(ctx, t,
		CommitFileAction{Action: PatchTextAction, Path: "lines.txt", SHA: fileSHA, Payload: []byte("2:3\x00two\n")},
	)
	require.Len(t, out.ChangedFiles, 1)
	require.Equal(t, "lines.txt", out.ChangedFiles[0].Path)
	require.Equal(t, "1\ntwo\n3", f.git(ctx, t, "show", "main:lines.txt"))

	// patch actions with different SHAs for one path are rejected.
	otherSHA := sha.Must(strings.Repeat("2", 40))
	_, err := f.svc.CommitFiles(ctx, &CommitFilesParams{
		WriteParams: f.write(), Branch: "main", Message: "patch",
		Actions: []CommitFileAction{
			{Action: PatchTextAction, Path: "lines.txt", SHA: fileSHA, Payload: []byte("1:2\x00x\n")},
			{Action: PatchTextAction, Path: "lines.txt", SHA: otherSHA, Payload: []byte("1:2\x00y\n")},
		},
	})
	require.Error(t, err)
	require.True(t, errors.IsInvalidArgument(err), "got %v", err)
}

func TestGc5CommitFilesNewBranchAndErrors(t *testing.T) {
	f := gc5NewFixture(t)
	ctx := t.Context()

	create := CommitFileAction{Action: CreateAction, Path: "x.txt", Payload: []byte("x\n")}

	t.Run("invalid params", func(t *testing.T) {
		_, err := f.svc.CommitFiles(ctx, &CommitFilesParams{})
		require.Error(t, err)
	})

	t.Run("default branch and new branch", func(t *testing.T) {
		out, err := f.svc.CommitFiles(ctx, &CommitFilesParams{
			WriteParams: f.write(), NewBranch: "refs/heads/feature-x", Message: "m",
			Actions: []CommitFileAction{create},
		})
		require.NoError(t, err)
		require.Equal(t, out.CommitID.String(), f.git(ctx, t, "rev-parse", "refs/heads/feature-x"))
	})

	t.Run("new branch already exists", func(t *testing.T) {
		_, err := f.svc.CommitFiles(ctx, &CommitFilesParams{
			WriteParams: f.write(), Branch: "main", NewBranch: "feature-x", Message: "m",
			Actions: []CommitFileAction{create},
		})
		require.True(t, errors.IsConflict(err), "got %v", err)
	})

	t.Run("invalid new branch name", func(t *testing.T) {
		_, err := f.svc.CommitFiles(ctx, &CommitFilesParams{
			WriteParams: f.write(), Branch: "main", NewBranch: "bad..name", Message: "m",
			Actions: []CommitFileAction{create},
		})
		require.True(t, errors.IsInvalidArgument(err), "got %v", err)
	})

	t.Run("missing source branch", func(t *testing.T) {
		_, err := f.svc.CommitFiles(ctx, &CommitFilesParams{
			WriteParams: f.write(), Branch: "nope", Message: "m", Actions: []CommitFileAction{create},
		})
		require.Error(t, err)
	})

	t.Run("no effective changes", func(t *testing.T) {
		_, err := f.svc.CommitFiles(ctx, &CommitFilesParams{
			WriteParams: f.write(), Branch: "main", Message: "m",
			Actions: []CommitFileAction{{Action: CreateAction, Path: "a.txt", Payload: []byte("a\n")}},
		})
		require.Error(t, err)
	})

	t.Run("invalid actions", func(t *testing.T) {
		tests := []struct {
			name   string
			action CommitFileAction
		}{
			{"empty path", CommitFileAction{Action: CreateAction, Path: "", Payload: []byte("x")}},
			{"unknown action", CommitFileAction{Action: "BOGUS", Path: "z.txt"}},
			{"create existing file", CommitFileAction{Action: CreateAction, Path: "a.txt", Payload: []byte("zz")}},
			{"delete missing file", CommitFileAction{Action: DeleteAction, Path: "missing.txt"}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := f.svc.CommitFiles(ctx, &CommitFilesParams{
					WriteParams: f.write(), Branch: "main", Message: "m", Actions: []CommitFileAction{tt.action},
				})
				require.Error(t, err)
			})
		}
	})

	t.Run("conflicting actions", func(t *testing.T) {
		_, err := f.svc.CommitFiles(ctx, &CommitFilesParams{
			WriteParams: f.write(), Branch: "main", Message: "m",
			Actions: []CommitFileAction{
				{Action: CreateAction, Path: "same.txt", Payload: []byte("1")},
				{Action: CreateAction, Path: "same.txt", Payload: []byte("2")},
			},
		})
		require.Error(t, err)
	})
}

func TestGc5CommitFilesEmptyRepo(t *testing.T) {
	svc := gc5NewService(t)
	ctx := t.Context()
	actor := Identity{Name: "Gc5Actor", Email: "gc5-actor@example.com"}

	newEmpty := func(t *testing.T) WriteParams {
		t.Helper()
		out, err := svc.CreateRepository(ctx, &CreateRepositoryParams{Actor: actor, DefaultBranch: "main"})
		require.NoError(t, err)
		return WriteParams{RepoUID: out.UID, Actor: actor}
	}

	t.Run("create files", func(t *testing.T) {
		out, err := svc.CommitFiles(ctx, &CommitFilesParams{
			WriteParams: newEmpty(t), Message: "init",
			Actions: []CommitFileAction{{Action: CreateAction, Path: "/dir/first.txt", Payload: []byte("1\n")}},
		})
		require.NoError(t, err)
		require.Len(t, out.ChangedFiles, 1)
		require.Equal(t, "dir/first.txt", out.ChangedFiles[0].Path)
	})

	t.Run("only create is allowed", func(t *testing.T) {
		_, err := svc.CommitFiles(ctx, &CommitFilesParams{
			WriteParams: newEmpty(t), Message: "init",
			Actions: []CommitFileAction{{Action: DeleteAction, Path: "x.txt"}},
		})
		require.True(t, errors.IsPreconditionFailed(err), "got %v", err)
	})

	t.Run("invalid path", func(t *testing.T) {
		_, err := svc.CommitFiles(ctx, &CommitFilesParams{
			WriteParams: newEmpty(t), Message: "init",
			Actions: []CommitFileAction{{Action: CreateAction, Path: "", Payload: []byte("x")}},
		})
		require.True(t, errors.IsInvalidArgument(err), "got %v", err)
	})

	t.Run("invalid new branch name", func(t *testing.T) {
		_, err := svc.CommitFiles(ctx, &CommitFilesParams{
			WriteParams: newEmpty(t), Branch: "main", NewBranch: "bad..name", Message: "init",
			Actions: []CommitFileAction{{Action: CreateAction, Path: "x.txt", Payload: []byte("x")}},
		})
		require.True(t, errors.IsInvalidArgument(err), "got %v", err)
	})
}

func TestGc5DiffVariants(t *testing.T) {
	f := gc5NewFixture(t)
	ctx := t.Context()

	base := f.head(ctx, t).String()
	f.commit(ctx, t,
		CommitFileAction{Action: UpdateAction, Path: "a.txt", SHA: sha.Must(f.git(ctx, t, "rev-parse", "main:a.txt")),
			Payload: []byte("a\nsecond line\n")},
		CommitFileAction{Action: CreateAction, Path: "added.txt", Payload: []byte("added\n")},
		CommitFileAction{Action: DeleteAction, Path: "dir/b.txt"},
	)
	head := f.head(ctx, t).String()

	t.Run("short stat", func(t *testing.T) {
		stat, err := f.svc.DiffShortStat(ctx, &DiffParams{ReadParams: f.read(), BaseRef: base, HeadRef: head})
		require.NoError(t, err)
		require.Equal(t, 3, stat.Files)
		require.Equal(t, 2, stat.Additions)
		require.Equal(t, 1, stat.Deletions)

		_, err = f.svc.DiffShortStat(ctx, &DiffParams{})
		require.Error(t, err)
		_, err = f.svc.DiffShortStat(ctx, &DiffParams{ReadParams: f.read(), BaseRef: base, HeadRef: "nope"})
		require.Error(t, err)
	})

	t.Run("file names", func(t *testing.T) {
		out, err := f.svc.DiffFileNames(ctx, &DiffParams{
			ReadParams: f.read(), BaseRef: base, HeadRef: head, MergeBase: true,
		})
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"a.txt", "added.txt", "dir/b.txt"}, out.Files)

		_, err = f.svc.DiffFileNames(ctx, &DiffParams{})
		require.Error(t, err)
		_, err = f.svc.DiffFileNames(ctx, &DiffParams{ReadParams: f.read(), BaseRef: base, HeadRef: "nope"})
		require.Error(t, err)
	})

	t.Run("diff with patches", func(t *testing.T) {
		chFiles, chErr := f.svc.Diff(ctx, &DiffParams{
			ReadParams: f.read(), BaseRef: base, HeadRef: head, IncludePatch: true,
		})
		statuses := map[string]enum.FileDiffStatus{}
		for fd := range chFiles {
			statuses[fd.Path] = fd.Status
			require.NotEmpty(t, fd.Patch)
		}
		require.NoError(t, <-chErr)
		require.Equal(t, enum.FileDiffStatusModified, statuses["a.txt"])
		require.Equal(t, enum.FileDiffStatusAdded, statuses["added.txt"])
		require.Equal(t, enum.FileDiffStatusDeleted, statuses["dir/b.txt"])
	})

	t.Run("diff cut", func(t *testing.T) {
		_, err := f.svc.DiffCut(ctx, &DiffCutParams{SourceCommitSHA: head, TargetCommitSHA: head})
		require.Error(t, err)

		out, err := f.svc.DiffCut(ctx, &DiffCutParams{
			ReadParams: f.read(), SourceCommitSHA: head, TargetCommitSHA: base, Path: "a.txt",
			LineStart: 2, LineStartNew: true, LineEnd: 2, LineEndNew: true,
		})
		require.NoError(t, err)
		require.Equal(t, base, out.MergeBaseSHA.String())
		require.Contains(t, strings.Join(out.Lines, "\n"), "second line")
		require.NotEmpty(t, out.LinesHeader)

		_, err = f.svc.DiffCut(ctx, &DiffCutParams{
			ReadParams: f.read(), SourceCommitSHA: head, TargetCommitSHA: base, Path: "nope.txt",
			LineStart: 1, LineStartNew: true, LineEnd: 1, LineEndNew: true,
		})
		require.Error(t, err)

		_, err = f.svc.DiffCut(ctx, &DiffCutParams{
			ReadParams: f.read(), SourceCommitSHA: "nope", TargetCommitSHA: base, Path: "a.txt",
		})
		require.Error(t, err)
	})
}

func TestGc5ParseFileDiffStatus(t *testing.T) {
	tests := []struct {
		in   diff.FileType
		want enum.FileDiffStatus
	}{
		{diff.FileAdd, enum.FileDiffStatusAdded},
		{diff.FileDelete, enum.FileDiffStatusDeleted},
		{diff.FileChange, enum.FileDiffStatusModified},
		{diff.FileRename, enum.FileDiffStatusRenamed},
		{diff.FileType(99), enum.FileDiffStatusUndefined},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, parseFileDiffStatus(tt.in))
	}
}

func TestGc5MapTreeNodeMode(t *testing.T) {
	tests := []struct {
		in   api.TreeNodeMode
		want TreeNodeMode
	}{
		{api.TreeNodeModeFile, TreeNodeModeFile},
		{api.TreeNodeModeExec, TreeNodeModeExec},
		{api.TreeNodeModeSymlink, TreeNodeModeSymlink},
		{api.TreeNodeModeCommit, TreeNodeModeCommit},
		{api.TreeNodeModeTree, TreeNodeModeTree},
	}
	for _, tt := range tests {
		got, err := mapTreeNodeMode(tt.in)
		require.NoError(t, err)
		require.Equal(t, tt.want, got)
	}

	_, err := mapTreeNodeMode(api.TreeNodeMode(-5))
	require.Error(t, err)
}

func TestGc5GetRefPath(t *testing.T) {
	tests := []struct {
		refType enum.RefType
		want    string
	}{
		{enum.RefTypeRaw, "refs/x"},
		{enum.RefTypeBranch, "refs/heads/refs/x"},
		{enum.RefTypeTag, "refs/tags/refs/x"},
		{enum.RefTypePullReqHead, "refs/pullreq/refs/x/head"},
		{enum.RefTypePullReqMerge, "refs/pullreq/refs/x/merge"},
		{enum.RefTypePullReqMergeQueue, "refs/mergequeue/refs/x"},
	}
	for _, tt := range tests {
		got, err := GetRefPath("refs/x", tt.refType)
		require.NoError(t, err, tt.refType)
		require.Equal(t, tt.want, got)
	}

	_, err := GetRefPath("x", enum.RefType(98765))
	require.Error(t, err)
}

func TestGc5ReferenceWalkPatterns(t *testing.T) {
	tests := []struct {
		name, base, query string
		want              []string
	}{
		{"nothing", "", "", []string{}},
		{"base only", "refs/tags", "", []string{"refs/tags/"}},
		{"exact", "refs/tags/", "^v1$", []string{"refs/tags/v1"}},
		{"suffix", "refs/tags", "v1$", []string{"refs/tags/**/*v1"}},
		{"prefix", "refs/tags", "^v1", []string{"refs/tags/v1*", "refs/tags/v1*/**"}},
		{"arbitrary", "refs/tags", "v1", []string{"refs/tags/**/*v1*", "refs/tags/**/*v1*/**"}},
		{"bad characters are stripped", "refs/tags", "v 1?[*]", []string{"refs/tags/**/*v1]*", "refs/tags/**/*v1]*/**"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, createReferenceWalkPatternsFromQuery(tt.base, tt.query))
		})
	}

	q, prefix, suffix := sanitizeReferenceQuery("")
	require.Empty(t, q)
	require.False(t, prefix)
	require.False(t, suffix)
}

type gc5Repo string

func (r gc5Repo) GetGitUID() string { return string(r) }

func TestGc5CreateReadParams(t *testing.T) {
	require.Equal(t, ReadParams{RepoUID: "uid-1"}, CreateReadParams(gc5Repo("uid-1")))
}

func TestGc5ParamsValidate(t *testing.T) {
	actor := Identity{Name: "N", Email: "n@example.com"}
	baseSHA := sha.Must(strings.Repeat("1", 40))

	require.Error(t, (&MergeParams{WriteParams: WriteParams{RepoUID: "r", Actor: actor}}).Validate())
	require.Error(t, (&MergeParams{
		WriteParams: WriteParams{RepoUID: "r", Actor: actor}, BaseBranch: "main",
	}).Validate())
	require.Error(t, (&MergeParams{
		WriteParams: WriteParams{RepoUID: "r", Actor: actor}, BaseBranch: "main", HeadBranch: "h",
		Refs: []RefUpdate{{}},
	}).Validate())
	require.NoError(t, (&MergeParams{
		WriteParams: WriteParams{RepoUID: "r", Actor: actor}, BaseSHA: baseSHA, HeadSHA: baseSHA,
		Refs: []RefUpdate{{Name: "refs/heads/x"}},
	}).Validate())

	require.Error(t, (&MergeVerifyParams{}).Validate())
	require.Error(t, (&MergeVerifyParams{ReadParams: ReadParams{RepoUID: "r"}}).Validate())
	require.Error(t, (&MergeVerifyParams{ReadParams: ReadParams{RepoUID: "r"}, BaseSHA: baseSHA}).Validate())
	require.NoError(t, (&MergeVerifyParams{
		ReadParams: ReadParams{RepoUID: "r"}, BaseSHA: baseSHA, HeadSHA: baseSHA,
	}).Validate())

	require.Error(t, (&MergeBaseParams{}).Validate())
	require.Error(t, (&MergeBaseParams{ReadParams: ReadParams{RepoUID: "r"}}).Validate())
	require.Error(t, (&MergeBaseParams{ReadParams: ReadParams{RepoUID: "r"}, Ref1: "a"}).Validate())
	require.NoError(t, (&MergeBaseParams{ReadParams: ReadParams{RepoUID: "r"}, Ref1: "a", Ref2: "b"}).Validate())
}

func TestGc5InfoRefsAndServicePack(t *testing.T) {
	f := gc5NewFixture(t)
	ctx := t.Context()

	var buf bytes.Buffer
	require.Error(t, f.svc.GetInfoRefs(ctx, &buf, &InfoRefsParams{}))

	require.NoError(t, f.svc.GetInfoRefs(ctx, &buf, &InfoRefsParams{
		ReadParams: f.read(), Service: "upload-pack", GitProtocol: "version=2",
	}))
	require.Contains(t, buf.String(), "# service=git-upload-pack")

	buf.Reset()
	require.NoError(t, f.svc.GetInfoRefs(ctx, &buf, &InfoRefsParams{ReadParams: f.read(), Service: "upload-pack"}))
	require.Contains(t, buf.String(), "refs/heads/main")

	require.Error(t, f.svc.GetInfoRefs(ctx, &buf, &InfoRefsParams{ReadParams: f.read(), Service: "bogus"}))

	// service pack validation.
	require.Error(t, f.svc.ServicePack(ctx, &ServicePackParams{}))
	require.Error(t, f.svc.ServicePack(ctx, &ServicePackParams{
		ReadParams:         &ReadParams{},
		ServicePackOptions: api.ServicePackOptions{Service: tenum.GitServiceTypeUploadPack},
	}))
	require.Error(t, f.svc.ServicePack(ctx, &ServicePackParams{
		WriteParams:        &WriteParams{},
		ServicePackOptions: api.ServicePackOptions{Service: tenum.GitServiceTypeReceivePack},
	}))
	require.Error(t, f.svc.ServicePack(ctx, &ServicePackParams{
		ServicePackOptions: api.ServicePackOptions{Service: tenum.GitServiceType("bogus")},
	}))

	// upload-pack asking for an unknown object is reported as an invalid argument.
	readParams := f.read()
	var stdout bytes.Buffer
	err := f.svc.ServicePack(ctx, &ServicePackParams{
		ReadParams: &readParams,
		ServicePackOptions: api.ServicePackOptions{
			Service:      tenum.GitServiceTypeUploadPack,
			StatelessRPC: true,
			Stdin:        strings.NewReader("0032want " + strings.Repeat("1", 40) + "\n00000009done\n"),
			Stdout:       &stdout,
			Stderr:       &bytes.Buffer{},
		},
	})
	require.Error(t, err)
}
