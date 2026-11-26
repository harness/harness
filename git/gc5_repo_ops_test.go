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
	"archive/tar"
	"bytes"
	"context"
	"io"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/hash"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/git/sharedrepo"
	"github.com/harness/gitness/git/types"

	"github.com/stretchr/testify/require"
)

type gc5HookClientFactory struct{}

func (gc5HookClientFactory) NewClient(map[string]string) (hook.Client, error) {
	return gc5HookClient{}, nil
}

type gc5HookClient struct{}

func (gc5HookClient) PreReceive(context.Context, hook.PreReceiveInput) (hook.Output, error) {
	return hook.Output{}, nil
}

func (gc5HookClient) Update(context.Context, hook.UpdateInput) (hook.Output, error) {
	return hook.Output{}, nil
}

func (gc5HookClient) PostReceive(context.Context, hook.PostReceiveInput) (hook.Output, error) {
	return hook.Output{}, nil
}

// gc5Fixture is a service with one repository on main containing a.txt, dir/b.txt and dir/sub/c.go.
type gc5Fixture struct {
	svc   *Service
	uid   string
	path  string
	actor Identity
}

func gc5NewService(t *testing.T) *Service {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available on PATH")
	}

	config := types.Config{Root: t.TempDir(), HookPath: "/path/to/githook"}
	adapter, err := api.New(config, api.NewInMemoryLastCommitCache(time.Minute), nil)
	require.NoError(t, err)
	runnerFactory, err := sharedrepo.NewRunnerFactory(config.Root)
	require.NoError(t, err)
	svc, err := New(config, adapter, gc5HookClientFactory{}, nil, runnerFactory)
	require.NoError(t, err)

	return svc
}

func gc5NewFixture(t *testing.T) *gc5Fixture {
	t.Helper()

	svc := gc5NewService(t)
	actor := Identity{Name: "Gc5Actor", Email: "gc5-actor@example.com"}
	out, err := svc.CreateRepository(t.Context(), &CreateRepositoryParams{
		Actor:         actor,
		DefaultBranch: "main",
		Files: []File{
			{Path: "a.txt", Content: []byte("a\n")},
			{Path: "dir/b.txt", Content: []byte("b\n")},
			{Path: "dir/sub/c.go", Content: []byte("package sub\n\nfunc C() int { return 1 }\n")},
		},
	})
	require.NoError(t, err)

	return &gc5Fixture{svc: svc, uid: out.UID, path: getFullPathForRepo(svc.reposRoot, out.UID), actor: actor}
}

func (f *gc5Fixture) read() ReadParams   { return ReadParams{RepoUID: f.uid} }
func (f *gc5Fixture) write() WriteParams { return WriteParams{RepoUID: f.uid, Actor: f.actor} }

func gc5GitIn(ctx context.Context, t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %v: %s", args, out)

	return strings.TrimSpace(string(out))
}

func (f *gc5Fixture) git(ctx context.Context, t *testing.T, args ...string) string {
	t.Helper()

	return gc5GitIn(ctx, t, f.path, args...)
}

func (f *gc5Fixture) head(ctx context.Context, t *testing.T) sha.SHA {
	t.Helper()

	return sha.Must(f.git(ctx, t, "rev-parse", "refs/heads/main"))
}

func (f *gc5Fixture) commit(ctx context.Context, t *testing.T, actions ...CommitFileAction) CommitFilesResponse {
	t.Helper()

	out, err := f.svc.CommitFiles(ctx, &CommitFilesParams{
		WriteParams: f.write(),
		Message:     "gc5 commit",
		Branch:      "main",
		Actions:     actions,
	})
	require.NoError(t, err)

	return out
}

func TestGc5DeleteRepository(t *testing.T) {
	f := gc5NewFixture(t)
	ctx := t.Context()

	require.Error(t, f.svc.DeleteRepository(ctx, &DeleteRepositoryParams{}))

	require.NoError(t, f.svc.DeleteRepository(ctx, &DeleteRepositoryParams{WriteParams: f.write()}))
	_, err := f.svc.GetDefaultBranch(ctx, &GetDefaultBranchParams{ReadParams: f.read()})
	require.Error(t, err)

	err = f.svc.DeleteRepository(ctx, &DeleteRepositoryParams{WriteParams: f.write()})
	require.True(t, errors.IsNotFound(err), "got %v", err)
}

func TestGc5DefaultBranch(t *testing.T) {
	f := gc5NewFixture(t)
	ctx := t.Context()

	_, err := f.svc.GetDefaultBranch(ctx, &GetDefaultBranchParams{})
	require.Error(t, err)

	out, err := f.svc.GetDefaultBranch(ctx, &GetDefaultBranchParams{ReadParams: f.read()})
	require.NoError(t, err)
	require.Equal(t, "main", out.BranchName)

	f.git(ctx, t, "branch", "develop", "main")

	require.Error(t, f.svc.UpdateDefaultBranch(ctx, &UpdateDefaultBranchParams{BranchName: "develop"}))
	require.Error(t, f.svc.UpdateDefaultBranch(ctx, &UpdateDefaultBranchParams{
		WriteParams: f.write(), BranchName: "bad..name",
	}))
	require.NoError(t, f.svc.UpdateDefaultBranch(ctx, &UpdateDefaultBranchParams{
		WriteParams: f.write(), BranchName: "develop",
	}))

	out, err = f.svc.GetDefaultBranch(ctx, &GetDefaultBranchParams{ReadParams: f.read()})
	require.NoError(t, err)
	require.Equal(t, "develop", out.BranchName)
}

func TestGc5GetRepositorySize(t *testing.T) {
	f := gc5NewFixture(t)

	out, err := f.svc.GetRepositorySize(t.Context(), &GetRepositorySizeParams{ReadParams: f.read()})
	require.NoError(t, err)
	require.Positive(t, out.Size)

	_, err = f.svc.GetRepositorySize(t.Context(), &GetRepositorySizeParams{
		ReadParams: ReadParams{RepoUID: "does-not-exist"},
	})
	require.Error(t, err)
}

func TestGc5HashRepository(t *testing.T) {
	f := gc5NewFixture(t)
	ctx := t.Context()

	params := &HashRepositoryParams{
		ReadParams:      f.read(),
		HashType:        hash.TypeSHA256,
		AggregationType: hash.AggregationTypeXOR,
	}

	first, err := f.svc.HashRepository(ctx, params)
	require.NoError(t, err)
	require.Len(t, first.Hash, 32)

	again, err := f.svc.HashRepository(ctx, params)
	require.NoError(t, err)
	require.Equal(t, first.Hash, again.Hash)

	f.git(ctx, t, "branch", "other", "main")
	changed, err := f.svc.HashRepository(ctx, params)
	require.NoError(t, err)
	require.NotEqual(t, first.Hash, changed.Hash)

	_, err = f.svc.HashRepository(ctx, &HashRepositoryParams{})
	require.Error(t, err)

	_, err = f.svc.HashRepository(ctx, &HashRepositoryParams{
		ReadParams: f.read(), HashType: "md5", AggregationType: hash.AggregationTypeXOR,
	})
	require.Error(t, err)

	_, err = f.svc.HashRepository(ctx, &HashRepositoryParams{
		ReadParams: ReadParams{RepoUID: "does-not-exist"}, HashType: hash.TypeSHA256,
		AggregationType: hash.AggregationTypeXOR,
	})
	require.Error(t, err)
}

func TestGc5Archive(t *testing.T) {
	f := gc5NewFixture(t)
	ctx := t.Context()

	require.Error(t, f.svc.Archive(ctx, ArchiveParams{}, io.Discard))
	require.Error(t, f.svc.Archive(ctx, ArchiveParams{ReadParams: f.read()}, io.Discard))
	require.Error(t, f.svc.Archive(ctx, ArchiveParams{
		ReadParams:    f.read(),
		ArchiveParams: api.ArchiveParams{Treeish: "main", Format: "rar"},
	}, io.Discard))
	require.Error(t, f.svc.Archive(ctx, ArchiveParams{
		ReadParams:    f.read(),
		ArchiveParams: api.ArchiveParams{Treeish: "no-such-branch"},
	}, io.Discard))

	var buf bytes.Buffer
	require.NoError(t, f.svc.Archive(ctx, ArchiveParams{
		ReadParams:    f.read(),
		ArchiveParams: api.ArchiveParams{Treeish: "main", Format: api.ArchiveFormatTar, Prefix: "pfx"},
	}, &buf))

	var names []string
	tr := tar.NewReader(&buf)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		names = append(names, hdr.Name)
	}
	require.Contains(t, names, "pfx/a.txt")
	require.Contains(t, names, "pfx/dir/b.txt")
}

func TestGc5ConvertRepoSource(t *testing.T) {
	svc := gc5NewService(t)

	require.Equal(t, "https://example.com/a/b.git", svc.convertRepoSource("https://example.com/a/b.git"))
	require.Equal(t, "/some/path", svc.convertRepoSource("/some/path"))
	require.Equal(t, getFullPathForRepo(svc.reposRoot, "abcdef"), svc.convertRepoSource("abcdef"))
}

func TestGc5SyncRepository(t *testing.T) {
	src := gc5NewFixture(t)
	ctx := t.Context()
	svc := src.svc

	dstUID, err := NewRepositoryUID()
	require.NoError(t, err)
	dst := WriteParams{RepoUID: dstUID, Actor: src.actor}

	_, err = svc.SyncRepository(ctx, &SyncRepositoryParams{})
	require.Error(t, err)

	_, err = svc.SyncRepository(ctx, &SyncRepositoryParams{WriteParams: dst, Source: src.uid})
	require.True(t, errors.IsNotFound(err), "got %v", err)

	out, err := svc.SyncRepository(ctx, &SyncRepositoryParams{
		WriteParams: dst, Source: src.uid, CreateIfNotExists: true,
	})
	require.NoError(t, err)
	require.Equal(t, "main", out.DefaultBranch)

	// second sync of an existing repo with an explicit default branch and refspec.
	src.git(ctx, t, "branch", "feature", "main")
	out, err = svc.SyncRepository(ctx, &SyncRepositoryParams{
		WriteParams:   dst,
		Source:        src.uid,
		RefSpecs:      []string{"+refs/heads/*:refs/heads/*"},
		DefaultBranch: "feature",
	})
	require.NoError(t, err)
	require.Equal(t, "feature", out.DefaultBranch)

	branch, err := svc.GetBranch(ctx, &GetBranchParams{
		ReadParams: ReadParams{RepoUID: dstUID}, BranchName: "feature",
	})
	require.NoError(t, err)
	require.Equal(t, src.head(ctx, t).String(), branch.Branch.SHA.String())

	// syncing from an invalid source fails.
	_, err = svc.SyncRepository(ctx, &SyncRepositoryParams{WriteParams: dst, Source: "nope"})
	require.Error(t, err)
}

func TestGc5SyncRepositoryEmptySource(t *testing.T) {
	svc := gc5NewService(t)
	ctx := t.Context()
	actor := Identity{Name: "Gc5Actor", Email: "gc5-actor@example.com"}

	srcOut, err := svc.CreateRepository(ctx, &CreateRepositoryParams{Actor: actor, DefaultBranch: "main"})
	require.NoError(t, err)

	dstUID, err := NewRepositoryUID()
	require.NoError(t, err)

	_, err = svc.SyncRepository(ctx, &SyncRepositoryParams{
		WriteParams: WriteParams{RepoUID: dstUID, Actor: actor}, Source: srcOut.UID, CreateIfNotExists: true,
	})
	require.Error(t, err)
}

func TestGc5GetRemoteDefaultBranch(t *testing.T) {
	f := gc5NewFixture(t)

	out, err := f.svc.GetRemoteDefaultBranch(t.Context(), &GetRemoteDefaultBranchParams{Source: f.uid})
	require.NoError(t, err)
	require.Equal(t, "main", out.BranchName)

	_, err = f.svc.GetRemoteDefaultBranch(t.Context(), &GetRemoteDefaultBranchParams{Source: "nope"})
	require.Error(t, err)
}

func TestGc5FetchObjects(t *testing.T) {
	src := gc5NewFixture(t)
	ctx := t.Context()
	svc := src.svc

	dstOut, err := svc.CreateRepository(ctx, &CreateRepositoryParams{Actor: src.actor, DefaultBranch: "main"})
	require.NoError(t, err)
	dst := WriteParams{RepoUID: dstOut.UID, Actor: src.actor}

	_, err = svc.FetchObjects(ctx, &FetchObjectsParams{})
	require.Error(t, err)

	head := src.head(ctx, t)
	_, err = svc.FetchObjects(ctx, &FetchObjectsParams{
		WriteParams: dst, Source: src.uid, ObjectSHAs: []sha.SHA{head},
	})
	require.NoError(t, err)

	dstPath := getFullPathForRepo(svc.reposRoot, dstOut.UID)
	require.Equal(t, "commit", gc5GitIn(ctx, t, dstPath, "cat-file", "-t", head.String()))

	_, err = svc.FetchObjects(ctx, &FetchObjectsParams{
		WriteParams: dst, Source: src.uid, ObjectSHAs: []sha.SHA{sha.Must(strings.Repeat("1", 40))},
	})
	require.Error(t, err)
}

func TestGc5SyncRefs(t *testing.T) {
	src := gc5NewFixture(t)
	ctx := t.Context()
	svc := src.svc

	dstUID, err := NewRepositoryUID()
	require.NoError(t, err)
	dst := WriteParams{RepoUID: dstUID, Actor: src.actor}
	_, err = svc.SyncRepository(ctx, &SyncRepositoryParams{WriteParams: dst, Source: src.uid, CreateIfNotExists: true})
	require.NoError(t, err)

	_, err = svc.SyncRefs(ctx, &SyncRefsParams{})
	require.Error(t, err)

	_, err = svc.SyncRefs(ctx, &SyncRefsParams{WriteParams: dst, Source: src.uid, Refs: []string{"refs/heads/missing"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "refs/heads/missing")

	// nothing changed yet.
	out, err := svc.SyncRefs(ctx, &SyncRefsParams{
		WriteParams: dst, Source: src.uid, Refs: []string{"refs/heads/main", "refs/heads/main"},
	})
	require.NoError(t, err)
	require.Empty(t, out.Refs)

	// new commit on source plus a new branch.
	src.commit(ctx, t, CommitFileAction{Action: CreateAction, Path: "new.txt", Payload: []byte("new\n")})
	src.git(ctx, t, "branch", "feature", "main")
	newHead := src.head(ctx, t)

	out, err = svc.SyncRefs(ctx, &SyncRefsParams{
		WriteParams: dst, Source: src.uid, Refs: []string{"refs/heads/main", "refs/heads/feature"},
	})
	require.NoError(t, err)
	require.Len(t, out.Refs, 2)

	byRef := map[string]hook.ReferenceUpdate{}
	for _, u := range out.Refs {
		byRef[u.Ref] = u
	}
	require.True(t, byRef["refs/heads/feature"].Old.IsNil())
	require.Equal(t, newHead, byRef["refs/heads/main"].New)

	branch, err := svc.GetBranch(ctx, &GetBranchParams{ReadParams: ReadParams{RepoUID: dstUID}, BranchName: "main"})
	require.NoError(t, err)
	require.Equal(t, newHead.String(), branch.Branch.SHA.String())
}

func TestGc5GetRepoLanguageStats(t *testing.T) {
	f := gc5NewFixture(t)
	ctx := t.Context()

	out, err := f.svc.GetRepoLanguageStats(ctx, &GetRepoLanguageStatsParams{ReadParams: f.read(), Branch: "main"})
	require.NoError(t, err)
	require.Contains(t, out.Stats, "Go")
	require.Positive(t, out.LOC)

	_, err = f.svc.GetRepoLanguageStats(ctx, &GetRepoLanguageStatsParams{ReadParams: f.read(), Branch: "nope"})
	require.Error(t, err)
}

func TestGc5Revert(t *testing.T) {
	f := gc5NewFixture(t)
	ctx := t.Context()

	base := f.head(ctx, t)
	f.commit(ctx, t, CommitFileAction{Action: CreateAction, Path: "extra.txt", Payload: []byte("extra\n")})
	tip := f.head(ctx, t)

	valid := RevertParams{
		WriteParams:     f.write(),
		ParentCommitSHA: tip,
		FromCommitSHA:   base,
		ToCommitSHA:     tip,
		RevertBranch:    "revert-extra",
		Message:         "Revert extra",
	}

	tests := []struct {
		name   string
		mutate func(p *RevertParams)
	}{
		{"missing repo", func(p *RevertParams) { p.RepoUID = "" }},
		{"missing message", func(p *RevertParams) { p.Message = "" }},
		{"missing branch", func(p *RevertParams) { p.RevertBranch = "" }},
		{"same commits", func(p *RevertParams) { p.FromCommitSHA = tip }},
		{"empty diff", func(p *RevertParams) { p.ToCommitSHA = base; p.FromCommitSHA = base }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := valid
			tt.mutate(&p)
			_, err := f.svc.Revert(ctx, &p)
			require.Error(t, err)
		})
	}

	// branching commits are rejected: "from" is a sibling of "to".
	side := sha.Must(f.git(ctx, t, "-c", "user.name=x", "-c", "user.email=x@example.com",
		"commit-tree", base.String()+"^{tree}", "-p", base.String(), "-m", "side"))

	p := valid
	p.FromCommitSHA = side
	_, err := f.svc.Revert(ctx, &p)
	require.Error(t, err)
	require.Contains(t, err.Error(), "branch out")

	author := Identity{Name: "Gc5Author", Email: "gc5-author@example.com"}
	when := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	p = valid
	p.Author, p.Committer = &author, &author
	p.AuthorDate, p.CommitterDate = &when, &when
	out, err := f.svc.Revert(ctx, &p)
	require.NoError(t, err)
	require.False(t, out.CommitSHA.IsEmpty())

	require.Equal(t, "Gc5Author", f.git(ctx, t, "log", "-1", "--format=%an", "refs/heads/revert-extra"))
	require.Equal(t, "Revert extra", f.git(ctx, t, "log", "-1", "--format=%s", "refs/heads/revert-extra"))
	require.Empty(t, f.git(ctx, t, "ls-tree", "--name-only", "refs/heads/revert-extra", "extra.txt"))

	// the revert branch exists now, so a second revert must fail.
	_, err = f.svc.Revert(ctx, &p)
	require.Error(t, err)
}
