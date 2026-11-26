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

package sharedrepo

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/git/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gtxFixture describes a source repository with a small commit graph:
//
//	base --- main (changes line 2 of a.txt)
//	   \---- clean (adds clean.txt)
//	   \---- side (changes line 2 of a.txt differently)
type gtxFixture struct {
	source string
	base   sha.SHA
	main   sha.SHA
	clean  sha.SHA
	side   sha.SHA
}

func gtxGit(ctx context.Context, t *testing.T, dir string, stdin []byte, args ...string) string {
	t.Helper()

	full := append([]string{
		"-c", "user.name=Gtx Tester",
		"-c", "user.email=gtx@example.com",
		"-c", "commit.gpgsign=false",
		"-c", "protocol.file.allow=always",
	}, args...)

	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_DATE=2024-01-02T03:04:05Z",
		"GIT_COMMITTER_DATE=2024-01-02T03:04:05Z",
	)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	require.NoError(t, cmd.Run(), "git %v failed: %s", args, stderr.String())

	return strings.TrimSpace(stdout.String())
}

func gtxWriteFile(t *testing.T, root, name, content string, mode os.FileMode) {
	t.Helper()

	full := filepath.Join(root, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o700))
	require.NoError(t, os.WriteFile(full, []byte(content), mode))
}

func gtxNewFixture(ctx context.Context, t *testing.T) gtxFixture {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available on PATH")
	}

	work := t.TempDir()
	gtxGit(ctx, t, work, nil, "init", "-q", "-b", "main")

	gtxWriteFile(t, work, "a.txt", "one\ntwo\nthree\n", 0o600)
	gtxWriteFile(t, work, "dir/b.txt", "bee\n", 0o600)
	gtxWriteFile(t, work, "run.sh", "#!/bin/sh\n", 0o700)
	require.NoError(t, os.Symlink("a.txt", filepath.Join(work, "link")))
	gtxGit(ctx, t, work, nil, "add", "-A")
	gtxGit(ctx, t, work, nil, "commit", "-q", "-m", "base")
	base := sha.Must(gtxGit(ctx, t, work, nil, "rev-parse", "HEAD"))

	gtxGit(ctx, t, work, nil, "checkout", "-q", "-b", "clean")
	gtxWriteFile(t, work, "clean.txt", "clean\n", 0o600)
	gtxGit(ctx, t, work, nil, "add", "-A")
	gtxGit(ctx, t, work, nil, "commit", "-q", "-m", "clean")
	clean := sha.Must(gtxGit(ctx, t, work, nil, "rev-parse", "HEAD"))

	gtxGit(ctx, t, work, nil, "checkout", "-q", "-b", "side", base.String())
	gtxWriteFile(t, work, "a.txt", "one\nTWO-side\nthree\n", 0o600)
	gtxGit(ctx, t, work, nil, "commit", "-q", "-a", "-m", "side")
	side := sha.Must(gtxGit(ctx, t, work, nil, "rev-parse", "HEAD"))

	gtxGit(ctx, t, work, nil, "checkout", "-q", "main")
	gtxWriteFile(t, work, "a.txt", "one\nTWO-main\nthree\n", 0o600)
	gtxGit(ctx, t, work, nil, "commit", "-q", "-a", "-m", "main")
	mainSHA := sha.Must(gtxGit(ctx, t, work, nil, "rev-parse", "HEAD"))

	source := filepath.Join(t.TempDir(), "source.git")
	gtxGit(ctx, t, work, nil, "clone", "-q", "--bare", work, source)

	return gtxFixture{source: source, base: base, main: mainSHA, clean: clean, side: side}
}

func gtxNewShared(ctx context.Context, t *testing.T, source string, alternates ...string) *SharedRepo {
	t.Helper()

	s, err := NewSharedRepo(t.TempDir(), source)
	require.NoError(t, err)
	t.Cleanup(func() { s.Close(ctx) })
	require.NoError(t, s.Init(ctx, alternates...))

	return s
}

func gtxSignature(name string) *api.Signature {
	return &api.Signature{
		Identity: api.Identity{Name: name, Email: strings.ToLower(name) + "@example.com"},
		When:     time.Date(2024, 3, 4, 5, 6, 7, 0, time.UTC),
	}
}

// gtxIndexListing returns the output of `git ls-files -s` of the shared repo as lines.
func gtxIndexListing(ctx context.Context, t *testing.T, s *SharedRepo) []string {
	t.Helper()

	out := gtxGit(ctx, t, s.Directory(), nil, "ls-files", "-s")
	if out == "" {
		return nil
	}

	return strings.Split(out, "\n")
}

func gtxIndexPaths(ctx context.Context, t *testing.T, s *SharedRepo) []string {
	t.Helper()

	var paths []string
	for _, line := range gtxIndexListing(ctx, t, s) {
		_, path, ok := strings.Cut(line, "\t")
		require.True(t, ok)
		paths = append(paths, path)
	}

	return paths
}

func TestGtxNewSharedRepo(t *testing.T) {
	ctx := context.Background()

	t.Run("empty source path", func(t *testing.T) {
		s, err := NewSharedRepo(t.TempDir(), "")
		require.Error(t, err)
		assert.Nil(t, s)
	})

	t.Run("temp dir is created and removed", func(t *testing.T) {
		base := filepath.Join(t.TempDir(), "nested", "base")
		s, err := NewSharedRepo(base, "/some/source")
		require.NoError(t, err)

		assert.DirExists(t, s.Directory())
		assert.Equal(t, base, filepath.Dir(s.Directory()))
		assert.True(t, strings.HasSuffix(s.Directory(), ".git") || strings.Contains(s.Directory(), ".git"))

		s.Close(ctx)
		assert.NoDirExists(t, s.Directory())

		// closing twice must be harmless
		s.Close(ctx)
	})

	t.Run("base dir cannot be created", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "file")
		require.NoError(t, os.WriteFile(file, nil, 0o600))

		_, err := NewSharedRepo(filepath.Join(file, "sub"), "/some/source")
		require.Error(t, err)
	})
}

func TestGtxSharedRepo_Init(t *testing.T) {
	ctx := context.Background()
	fx := gtxNewFixture(ctx, t)

	t.Run("alternates", func(t *testing.T) {
		s := gtxNewShared(ctx, t, fx.source, "/gtx/extra/one", "/gtx/extra/two")

		content, err := os.ReadFile(filepath.Join(s.Directory(), "objects", "info", "alternates"))
		require.NoError(t, err)

		assert.Equal(t,
			[]string{filepath.Join(fx.source, "objects"), "/gtx/extra/one", "/gtx/extra/two"},
			strings.Split(strings.TrimSpace(string(content)), "\n"))
	})

	t.Run("objects of source are visible", func(t *testing.T) {
		s := gtxNewShared(ctx, t, fx.source)

		assert.Equal(t, "commit", gtxGit(ctx, t, s.Directory(), nil, "cat-file", "-t", fx.main.String()))
	})

	t.Run("fails if directory is gone", func(t *testing.T) {
		s, err := NewSharedRepo(t.TempDir(), fx.source)
		require.NoError(t, err)
		s.Close(ctx)

		require.Error(t, s.Init(ctx))
	})

	t.Run("fails if alternates file cannot be created", func(t *testing.T) {
		s, err := NewSharedRepo(t.TempDir(), fx.source)
		require.NoError(t, err)
		t.Cleanup(func() { s.Close(ctx) })

		// a regular file where the objects directory belongs makes the alternates path unusable
		gtxWriteFile(t, s.Directory(), "objects", "not a directory", 0o600)

		require.Error(t, s.Init(ctx))
	})
}

func TestGtxSharedRepo_IndexOperations(t *testing.T) {
	ctx := context.Background()
	fx := gtxNewFixture(ctx, t)
	s := gtxNewShared(ctx, t, fx.source)

	treeSHA, err := s.GetTreeSHA(ctx, fx.main.String())
	require.NoError(t, err)
	assert.Equal(t, gtxGit(ctx, t, fx.source, nil, "rev-parse", fx.main.String()+"^{tree}"), treeSHA.String())

	t.Run("set index from tree and list files", func(t *testing.T) {
		require.NoError(t, s.SetIndex(ctx, treeSHA))

		all, err := s.LsFiles(ctx)
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"a.txt", "dir/b.txt", "link", "run.sh"}, gtxNonEmpty(all))

		some, err := s.LsFiles(ctx, "a.txt", "nonexistent.txt")
		require.NoError(t, err)
		assert.Equal(t, []string{"a.txt"}, gtxNonEmpty(some))

		none, err := s.LsFiles(ctx, "nonexistent.txt")
		require.NoError(t, err)
		assert.Empty(t, gtxNonEmpty(none))
	})

	t.Run("set index with unknown treeish", func(t *testing.T) {
		unknown := sha.Must("1234567890123456789012345678901234567890")
		require.Error(t, s.SetIndex(ctx, unknown))
	})

	t.Run("remove files from index", func(t *testing.T) {
		require.NoError(t, s.SetIndex(ctx, treeSHA))

		// empty names are ignored and unknown names are no-ops.
		require.NoError(t, s.RemoveFilesFromIndex(ctx, "a.txt", "", "dir/b.txt", "gtx-unknown.txt"))
		assert.ElementsMatch(t, []string{"link", "run.sh"}, gtxIndexPaths(ctx, t, s))
	})

	t.Run("clear index", func(t *testing.T) {
		require.NoError(t, s.SetIndex(ctx, treeSHA))
		require.NoError(t, s.ClearIndex(ctx))
		assert.Empty(t, gtxIndexListing(ctx, t, s))
	})

	t.Run("default index requires HEAD", func(t *testing.T) {
		require.Error(t, s.SetDefaultIndex(ctx), "shared repo has no HEAD commit yet")

		gtxGit(ctx, t, s.Directory(), nil, "symbolic-ref", "HEAD", "refs/heads/gtx")
		gtxGit(ctx, t, s.Directory(), nil, "update-ref", "refs/heads/gtx", fx.main.String())

		require.NoError(t, s.SetDefaultIndex(ctx))
		assert.ElementsMatch(t, []string{"a.txt", "dir/b.txt", "link", "run.sh"}, gtxIndexPaths(ctx, t, s))
	})

	t.Run("write tree", func(t *testing.T) {
		require.NoError(t, s.SetIndex(ctx, treeSHA))

		got, err := s.WriteTree(ctx)
		require.NoError(t, err)
		assert.Equal(t, treeSHA, got)
	})

	t.Run("write tree fails for unmerged index", func(t *testing.T) {
		require.NoError(t, s.ClearIndex(ctx))

		blob, err := s.WriteGitObject(ctx, strings.NewReader("conflict\n"))
		require.NoError(t, err)

		// stage 1 and 2 only: an unmerged index can't be written as a tree.
		gtxGit(ctx, t, s.Directory(), []byte("100644 "+blob.String()+" 1\tu.txt\n"+
			"100644 "+blob.String()+" 2\tu.txt\n"), "update-index", "--index-info")

		_, err = s.WriteTree(ctx)
		require.Error(t, err)
	})
}

func gtxNonEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v != "" {
			out = append(out, v)
		}
	}

	return out
}

func TestGtxSharedRepo_ObjectOperations(t *testing.T) {
	ctx := context.Background()
	fx := gtxNewFixture(ctx, t)
	s := gtxNewShared(ctx, t, fx.source)

	t.Run("write git object", func(t *testing.T) {
		blob, err := s.WriteGitObject(ctx, strings.NewReader("hello\n"))
		require.NoError(t, err)
		assert.Equal(t, gtxGit(ctx, t, s.Directory(), []byte("hello\n"), "hash-object", "--stdin"), blob.String())
		assert.Equal(t, "hello", gtxGit(ctx, t, s.Directory(), nil, "cat-file", "-p", blob.String()))
	})

	t.Run("show file", func(t *testing.T) {
		tests := []struct {
			name     string
			path     string
			rev      string
			want     string
			wantFail bool
		}{
			{name: "existing file", path: "a.txt", rev: fx.base.String(), want: "one\ntwo\nthree\n"},
			{name: "nested file", path: "dir/b.txt", rev: fx.main.String(), want: "bee\n"},
			{name: "whitespace is trimmed", path: " a.txt ", rev: " " + fx.side.String() + " ", want: "one\nTWO-side\nthree\n"},
			{name: "missing file", path: "gtx-missing.txt", rev: fx.base.String(), wantFail: true},
			{name: "unknown rev", path: "a.txt", rev: "gtx-unknown-rev", wantFail: true},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				var out bytes.Buffer
				err := s.ShowFile(ctx, test.path, test.rev, &out)
				if test.wantFail {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, test.want, out.String())
			})
		}
	})

	t.Run("get tree sha", func(t *testing.T) {
		got, err := s.GetTreeSHA(ctx, fx.clean.String())
		require.NoError(t, err)
		assert.Equal(t, gtxGit(ctx, t, fx.source, nil, "rev-parse", fx.clean.String()+"^{tree}"), got.String())

		_, err = s.GetTreeSHA(ctx, "gtx-unknown-rev")
		require.Error(t, err)
		assert.True(t, errors.IsNotFound(err), "unexpected error: %v", err)
	})

	t.Run("add object to index", func(t *testing.T) {
		require.NoError(t, s.ClearIndex(ctx))

		blob, err := s.WriteGitObject(ctx, strings.NewReader("content\n"))
		require.NoError(t, err)

		require.NoError(t, s.AddObjectToIndex(ctx, "100644", blob, "x/y.txt"))
		require.NoError(t, s.AddObjectToIndex(ctx, "100755", blob, "x/z.sh"))
		assert.Equal(t, []string{
			"100644 " + blob.String() + " 0\tx/y.txt",
			"100755 " + blob.String() + " 0\tx/z.sh",
		}, gtxIndexListing(ctx, t, s))

		err = s.AddObjectToIndex(ctx, "100644", blob, ".git/config")
		require.Error(t, err)
		assert.True(t, errors.IsInvalidArgument(err), "unexpected error: %v", err)

		err = s.AddObjectToIndex(ctx, "gtx-invalid-mode", blob, "q.txt")
		require.Error(t, err)
	})

	t.Run("diff", func(t *testing.T) {
		var out bytes.Buffer
		require.NoError(t, s.WriteDiff(ctx, fx.base.String(), fx.main.String(), &out))
		assert.Contains(t, out.String(), "-two\n+TWO-main\n")

		out.Reset()
		require.NoError(t, s.WriteDiff(ctx, fx.base.String(), fx.base.String(), &out))
		assert.Empty(t, out.String())

		require.Error(t, s.WriteDiff(ctx, "gtx-unknown-rev", fx.main.String(), &out))
	})

	t.Run("merge base", func(t *testing.T) {
		got, err := s.MergeBase(ctx, fx.main.String(), fx.side.String())
		require.NoError(t, err)
		assert.Equal(t, fx.base.String(), got)

		_, err = s.MergeBase(ctx, fx.main.String(), "gtx-unknown-rev")
		require.Error(t, err)
	})

	t.Run("commit sha list for rebase", func(t *testing.T) {
		got, err := s.CommitSHAsForRebase(ctx, fx.main, fx.clean)
		require.NoError(t, err)
		assert.Equal(t, []sha.SHA{fx.clean}, got)

		got, err = s.CommitSHAsForRebase(ctx, fx.main, fx.main)
		require.NoError(t, err)
		assert.Empty(t, got)

		_, err = s.CommitSHAsForRebase(ctx, fx.main, sha.Must("1234567890123456789012345678901234567890"))
		require.Error(t, err)
	})
}

func TestGtxSharedRepo_MergeTree(t *testing.T) {
	ctx := context.Background()
	fx := gtxNewFixture(ctx, t)
	s := gtxNewShared(ctx, t, fx.source)

	t.Run("clean merge with explicit merge base", func(t *testing.T) {
		tree, conflicts, err := s.MergeTree(ctx, fx.base, fx.main, fx.clean)
		require.NoError(t, err)
		assert.Empty(t, conflicts)
		assert.False(t, tree.IsEmpty())

		files := gtxGit(ctx, t, s.Directory(), nil, "ls-tree", "-r", "--name-only", tree.String())
		assert.ElementsMatch(t, []string{"a.txt", "clean.txt", "dir/b.txt", "link", "run.sh"}, strings.Split(files, "\n"))
	})

	t.Run("clean merge with computed merge base", func(t *testing.T) {
		tree, conflicts, err := s.MergeTree(ctx, sha.None, fx.main, fx.clean)
		require.NoError(t, err)
		assert.Empty(t, conflicts)
		assert.False(t, tree.IsEmpty())
	})

	t.Run("conflicting merge", func(t *testing.T) {
		tree, conflicts, err := s.MergeTree(ctx, fx.base, fx.main, fx.side)
		require.NoError(t, err)
		assert.Equal(t, []string{"a.txt"}, conflicts)
		assert.False(t, tree.IsEmpty())
	})

	t.Run("invalid commit", func(t *testing.T) {
		unknown := sha.Must("1234567890123456789012345678901234567890")
		_, _, err := s.MergeTree(ctx, sha.None, fx.main, unknown)
		require.Error(t, err)
	})
}

func TestGtxCleanupMergeConflicts(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "nil", in: nil, want: []string{}},
		{name: "blank entries are dropped", in: []string{"", "  ", "\t"}, want: []string{}},
		{
			name: "entries are trimmed", in: []string{" a.txt ", "b.txt\n", "", "c/d.txt"},
			want: []string{"a.txt", "b.txt", "c/d.txt"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, CleanupMergeConflicts(test.in))
		})
	}
}

func TestGtxSharedRepo_CommitTree(t *testing.T) {
	ctx := context.Background()
	fx := gtxNewFixture(ctx, t)
	s := gtxNewShared(ctx, t, fx.source)

	tree, err := s.GetTreeSHA(ctx, fx.main.String())
	require.NoError(t, err)

	author := gtxSignature("Author")
	committer := gtxSignature("Committer")

	tests := []struct {
		name        string
		message     string
		signoff     bool
		parents     []sha.SHA
		wantTrailer string
	}{
		{name: "root commit", message: "root commit"},
		{name: "single parent", message: "child commit", parents: []sha.SHA{fx.main}},
		{name: "merge commit", message: "merge commit", parents: []sha.SHA{fx.main, fx.clean}},
		{
			name: "signed off", message: "signed", signoff: true, parents: []sha.SHA{fx.base},
			wantTrailer: "Signed-off-by: Committer <committer@example.com>",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			commit, err := s.CommitTree(ctx, author, committer, tree, test.message, test.signoff, test.parents...)
			require.NoError(t, err)

			raw := gtxGit(ctx, t, s.Directory(), nil, "cat-file", "-p", commit.String())
			assert.Contains(t, raw, "tree "+tree.String()+"\n")
			assert.Contains(t, raw, "author Author <author@example.com> ")
			assert.Contains(t, raw, "committer Committer <committer@example.com> ")
			assert.Contains(t, raw, "\n\n"+test.message)
			assert.Equal(t, len(test.parents), strings.Count(raw, "\nparent ")+boolToInt(strings.HasPrefix(raw, "parent ")))
			for _, p := range test.parents {
				assert.Contains(t, raw, "parent "+p.String())
			}
			if test.wantTrailer != "" {
				assert.Contains(t, raw, test.wantTrailer)
			} else {
				assert.NotContains(t, raw, "Signed-off-by")
			}
		})
	}

	t.Run("unknown tree", func(t *testing.T) {
		unknown := sha.Must("1234567890123456789012345678901234567890")
		_, err := s.CommitTree(ctx, author, committer, unknown, "msg", false)
		require.Error(t, err)
	})
}

func boolToInt(b bool) int {
	if b {
		return 1
	}

	return 0
}

func TestGtxSharedRepo_ApplyToIndex(t *testing.T) {
	ctx := context.Background()
	fx := gtxNewFixture(ctx, t)
	s := gtxNewShared(ctx, t, fx.source)

	baseTree, err := s.GetTreeSHA(ctx, fx.base.String())
	require.NoError(t, err)
	require.NoError(t, s.SetIndex(ctx, baseTree))

	var patch bytes.Buffer
	require.NoError(t, s.WriteDiff(ctx, fx.base.String(), fx.main.String(), &patch))

	patchFile := filepath.Join(t.TempDir(), "main.patch")
	require.NoError(t, os.WriteFile(patchFile, patch.Bytes(), 0o600))

	require.NoError(t, s.ApplyToIndex(ctx, patchFile))

	got, err := s.WriteTree(ctx)
	require.NoError(t, err)
	wantTree, err := s.GetTreeSHA(ctx, fx.main.String())
	require.NoError(t, err)
	assert.Equal(t, wantTree, got)

	// the same patch can't be applied twice.
	err = s.ApplyToIndex(ctx, patchFile)
	require.Error(t, err)
	assert.True(t, errors.IsConflict(err), "unexpected error: %v", err)

	// a missing patch file is reported as an error, too.
	require.Error(t, s.ApplyToIndex(ctx, filepath.Join(t.TempDir(), "missing.patch")))
}

func TestGtxSharedRepo_FileOperations(t *testing.T) {
	ctx := context.Background()
	fx := gtxNewFixture(ctx, t)

	baseTree := func(t *testing.T, s *SharedRepo) sha.SHA {
		t.Helper()
		tree, err := s.GetTreeSHA(ctx, fx.base.String())
		require.NoError(t, err)
		require.NoError(t, s.SetIndex(ctx, tree))
		return tree
	}
	blobOf := func(t *testing.T, s *SharedRepo, path string) string {
		t.Helper()
		tree, err := s.WriteTree(ctx)
		require.NoError(t, err)
		return gtxGit(ctx, t, s.Directory(), nil, "cat-file", "-p", tree.String()+":"+path)
	}
	modeOf := func(t *testing.T, s *SharedRepo, path string) string {
		t.Helper()
		out := gtxGit(ctx, t, s.Directory(), nil, "ls-files", "-s", "--", path)
		require.NotEmpty(t, out)
		return strings.Fields(out)[0]
	}
	nodeSHA := func(t *testing.T, s *SharedRepo, path string) sha.SHA {
		t.Helper()
		node, err := api.GetTreeNode(ctx, s.Directory(), fx.base.String(), path, false)
		require.NoError(t, err)
		return node.SHA
	}

	t.Run("create file", func(t *testing.T) {
		tests := []struct {
			name     string
			path     string
			useBase  bool
			wantErr  func(error) bool
			wantFile bool
		}{
			{name: "new file", path: "new.txt", useBase: true, wantFile: true},
			{name: "new nested file", path: "dir/sub/new.txt", useBase: true, wantFile: true},
			{name: "no treeish skips availability check", path: "new.txt", useBase: false, wantFile: true},
			{name: "existing file", path: "a.txt", useBase: true, wantErr: errors.IsConflict},
			{name: "existing directory", path: "dir", useBase: true, wantErr: errors.IsConflict},
			{name: "existing symlink", path: "link", useBase: true, wantErr: errors.IsConflict},
			{name: "file in place of parent dir", path: "a.txt/inner.txt", useBase: true, wantErr: errors.IsConflict},
			{name: "invalid path", path: ".git/config", useBase: true, wantErr: errors.IsInvalidArgument},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				s := gtxNewShared(ctx, t, fx.source)
				tree := baseTree(t, s)
				if !test.useBase {
					tree = sha.None
				}

				blob, err := s.CreateFile(ctx, tree, test.path, "100644", []byte("created\n"))
				if test.wantErr != nil {
					require.Error(t, err)
					assert.True(t, test.wantErr(err), "unexpected error: %v", err)
					return
				}

				require.NoError(t, err)
				assert.False(t, blob.IsEmpty())
				if test.wantFile {
					assert.Equal(t, "created", blobOf(t, s, test.path))
				}
			})
		}
	})

	t.Run("update file", func(t *testing.T) {
		tests := []struct {
			name     string
			path     string
			useSHA   bool
			wrongSHA bool
			wantMode string
			wantErr  func(error) bool
		}{
			{name: "regular file", path: "a.txt", wantMode: "100644"},
			{name: "with matching sha", path: "a.txt", useSHA: true, wantMode: "100644"},
			{name: "executable keeps mode", path: "run.sh", wantMode: "100755"},
			{name: "sha mismatch", path: "a.txt", wrongSHA: true, wantErr: errors.IsInvalidArgument},
			{name: "missing file", path: "gtx-missing.txt", wantErr: errors.IsNotFound},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				s := gtxNewShared(ctx, t, fx.source)
				tree := baseTree(t, s)

				objectSHA := sha.None
				switch {
				case test.wrongSHA:
					objectSHA = sha.Must("1234567890123456789012345678901234567890")
				case test.useSHA:
					objectSHA = nodeSHA(t, s, test.path)
				}

				blob, err := s.UpdateFile(ctx, tree, test.path, objectSHA, "100644", []byte("updated\n"))
				if test.wantErr != nil {
					require.Error(t, err)
					assert.True(t, test.wantErr(err), "unexpected error: %v", err)
					return
				}

				require.NoError(t, err)
				assert.False(t, blob.IsEmpty())
				assert.Equal(t, "updated", blobOf(t, s, test.path))
				assert.Equal(t, test.wantMode, modeOf(t, s, test.path))
			})
		}
	})

	t.Run("move file", func(t *testing.T) {
		tests := []struct {
			name        string
			path        string
			payload     string
			wantNewPath string
			wantContent string
			wantMode    string
			wantErr     func(error) bool
		}{
			{
				name: "rename only", path: "a.txt", payload: "renamed.txt",
				wantNewPath: "renamed.txt", wantContent: "one\ntwo\nthree", wantMode: "100644",
			},
			{
				name: "rename into new dir", path: "dir/b.txt", payload: "/other/../x/b.txt",
				wantNewPath: "x/b.txt", wantContent: "bee", wantMode: "100644",
			},
			{
				name: "rename and replace content", path: "a.txt", payload: "renamed.txt\x00moved\n",
				wantNewPath: "renamed.txt", wantContent: "moved", wantMode: "100644",
			},
			{
				name: "executable keeps mode", path: "run.sh", payload: "bin/run.sh\x00#!/bin/bash\n",
				wantNewPath: "bin/run.sh", wantContent: "#!/bin/bash", wantMode: "100755",
			},
			{name: "empty target", path: "a.txt", payload: "", wantErr: func(err error) bool {
				return errors.Is(err, api.ErrInvalidPath)
			}},
			{name: "target in .git", path: "a.txt", payload: ".git/hooks", wantErr: func(err error) bool {
				return errors.Is(err, api.ErrInvalidPath)
			}},
			{name: "missing source", path: "gtx-missing.txt", payload: "x.txt", wantErr: errors.IsNotFound},
			{name: "target exists", path: "a.txt", payload: "run.sh", wantErr: errors.IsConflict},
			{name: "target is directory", path: "a.txt", payload: "dir", wantErr: errors.IsConflict},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				s := gtxNewShared(ctx, t, fx.source)
				tree := baseTree(t, s)

				newPath, blob, err := s.MoveFile(ctx, tree, test.path, sha.None, "100644", []byte(test.payload))
				if test.wantErr != nil {
					require.Error(t, err)
					assert.True(t, test.wantErr(err), "unexpected error: %v", err)
					return
				}

				require.NoError(t, err)
				assert.Equal(t, test.wantNewPath, newPath)
				assert.False(t, blob.IsEmpty())
				assert.Equal(t, test.wantContent, blobOf(t, s, newPath))
				assert.Equal(t, test.wantMode, modeOf(t, s, newPath))
				assert.NotContains(t, gtxIndexPaths(ctx, t, s), test.path)
			})
		}
	})

	t.Run("delete file", func(t *testing.T) {
		s := gtxNewShared(ctx, t, fx.source)
		baseTree(t, s)

		require.NoError(t, s.DeleteFile(ctx, "a.txt"))
		assert.NotContains(t, gtxIndexPaths(ctx, t, s), "a.txt")

		err := s.DeleteFile(ctx, "a.txt")
		require.Error(t, err)
		assert.True(t, errors.IsNotFound(err), "unexpected error: %v", err)

		err = s.DeleteFile(ctx, "gtx-missing.txt")
		require.Error(t, err)
		assert.True(t, errors.IsNotFound(err), "unexpected error: %v", err)
	})

	t.Run("patch text file", func(t *testing.T) {
		tests := []struct {
			name     string
			path     string
			payloads []string
			want     string
			wantErr  func(error) bool
		}{
			{name: "replace middle line", path: "a.txt", payloads: []string{"2:3\x00TWO\n"}, want: "one\nTWO\nthree"},
			{
				name: "insert before first line", path: "a.txt", payloads: []string{"1:1\x00zero\n"},
				want: "zero\none\ntwo\nthree",
			},
			{name: "append at eof", path: "a.txt", payloads: []string{"eof:eof\x00four\n"}, want: "one\ntwo\nthree\nfour"},
			{
				name: "two replacements", path: "a.txt",
				payloads: []string{"3:4\x00THREE\n", "1:2\x00ONE\n"}, want: "ONE\ntwo\nTHREE",
			},
			{name: "delete tail", path: "a.txt", payloads: []string{"2:eof\x00"}, want: "one"},
			{
				name: "overlapping ranges", path: "a.txt", payloads: []string{"1:3\x00x\n", "2:4\x00y\n"},
				wantErr: errors.IsInvalidArgument,
			},
			{name: "beyond eof", path: "a.txt", payloads: []string{"2:20\x00x\n"}, wantErr: errors.IsInvalidArgument},
			{name: "bad payload", path: "a.txt", payloads: []string{"nope"}, wantErr: errors.IsInvalidArgument},
			{name: "missing file", path: "gtx-missing.txt", payloads: []string{"1:1\x00x\n"}, wantErr: errors.IsNotFound},
		}

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				s := gtxNewShared(ctx, t, fx.source)
				tree := baseTree(t, s)

				payloads := make([][]byte, len(test.payloads))
				for i := range test.payloads {
					payloads[i] = []byte(test.payloads[i])
				}

				blob, err := s.PatchTextFile(ctx, tree, test.path, sha.None, payloads)
				if test.wantErr != nil {
					require.Error(t, err)
					assert.True(t, test.wantErr(err), "unexpected error: %v", err)
					return
				}

				require.NoError(t, err)
				assert.False(t, blob.IsEmpty())
				assert.Equal(t, test.want, blobOf(t, s, test.path))
			})
		}
	})
}

func TestGtxParseMovePayload(t *testing.T) {
	tests := []struct {
		name        string
		payload     string
		wantPath    string
		wantContent []byte
		wantErr     bool
	}{
		{name: "path only", payload: "a/b.txt", wantPath: "a/b.txt"},
		{name: "path is normalized", payload: "/a/./c/../b.txt", wantPath: "a/b.txt"},
		{name: "path and content", payload: "a.txt\x00data", wantPath: "a.txt", wantContent: []byte("data")},
		{name: "empty content", payload: "a.txt\x00", wantPath: "a.txt", wantContent: []byte{}},
		{name: "content with zero bytes", payload: "a.txt\x00x\x00y", wantPath: "a.txt", wantContent: []byte("x\x00y")},
		{name: "empty path", payload: "", wantErr: true},
		{name: "root path", payload: "/", wantErr: true},
		{name: "git dir", payload: "x/.GIT/y", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path, content, err := parseMovePayload([]byte(test.payload))
			if test.wantErr {
				require.ErrorIs(t, err, api.ErrInvalidPath)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.wantPath, path)
			assert.Equal(t, test.wantContent, content)
		})
	}
}

func TestGtxFilePriority(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{name: "0123456789abcdef0123456789abcdef01234567", want: 0},
		{name: "pack-1.keep", want: 1},
		{name: "pack-1.pack", want: 2},
		{name: "pack-1.rev", want: 3},
		{name: "pack-1.idx", want: 4},
		{name: "pack-1.bitmap", want: 5},
		{name: "pack-1", want: 5},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, filePriority(test.name))
		})
	}
}

func TestGtxSharedRepo_MoveObjectsErrors(t *testing.T) {
	ctx := context.Background()
	fx := gtxNewFixture(ctx, t)

	t.Run("no source repository", func(t *testing.T) {
		s := &SharedRepo{repoPath: t.TempDir()}
		moved, err := s.MoveObjects(ctx)
		require.Error(t, err)
		assert.Nil(t, moved)
	})

	t.Run("destination does not exist", func(t *testing.T) {
		s := &SharedRepo{repoPath: t.TempDir(), sourceRepoPath: filepath.Join(t.TempDir(), "gone")}
		_, err := s.MoveObjects(ctx)
		require.Error(t, err)
	})

	t.Run("destination is not a directory", func(t *testing.T) {
		src := t.TempDir()
		gtxWriteFile(t, src, "objects", "file", 0o600)

		s := &SharedRepo{repoPath: t.TempDir(), sourceRepoPath: src}
		_, err := s.MoveObjects(ctx)
		require.Error(t, err)
	})

	t.Run("shared repo directory is gone", func(t *testing.T) {
		src := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(src, "objects"), 0o700))

		s := &SharedRepo{repoPath: filepath.Join(t.TempDir(), "gone"), sourceRepoPath: src}
		_, err := s.MoveObjects(ctx)
		require.Error(t, err)

		_, err = s.Files()
		require.Error(t, err)
	})

	t.Run("object cannot be moved or copied", func(t *testing.T) {
		s := gtxNewShared(ctx, t, fx.source)
		blob, err := s.WriteGitObject(ctx, strings.NewReader("gtx object\n"))
		require.NoError(t, err)

		// a directory at the object's target path makes both the rename and the copy fail.
		name := blob.String()
		require.NoError(t, os.MkdirAll(filepath.Join(fx.source, "objects", name[:2], name[2:]), 0o700))

		moved, err := s.MoveObjects(ctx)
		require.Error(t, err)
		assert.Nil(t, moved)
	})

	t.Run("files are listed by priority", func(t *testing.T) {
		s := gtxNewShared(ctx, t, fx.source)
		blob, err := s.WriteGitObject(ctx, strings.NewReader("gtx files\n"))
		require.NoError(t, err)

		packDir := filepath.Join(s.Directory(), "objects", "pack")
		require.NoError(t, os.MkdirAll(packDir, 0o700))
		for _, name := range []string{"pack-x.idx", "pack-x.pack", "pack-x.rev", "pack-x.keep"} {
			gtxWriteFile(t, packDir, name, "x", 0o600)
		}

		files, err := s.Files()
		require.NoError(t, err)
		require.Len(t, files, 5)
		assert.Equal(t, blob.String()[:2]+"/"+blob.String()[2:], files[0])
		assert.Equal(t, []string{"pack/pack-x.keep", "pack/pack-x.pack", "pack/pack-x.rev", "pack/pack-x.idx"}, files[1:])
	})
}

func TestGtxSharedRepo_FetchObjects(t *testing.T) {
	ctx := context.Background()
	fx := gtxNewFixture(ctx, t)

	// the target has none of the objects of the fixture
	target := t.TempDir()
	gtxGit(ctx, t, target, nil, "init", "-q", "--bare")

	s, err := NewSharedRepo(t.TempDir(), target)
	require.NoError(t, err)
	t.Cleanup(func() { s.Close(ctx) })
	// no alternates to the fixture, so objects must really be fetched.
	gtxGit(ctx, t, s.Directory(), nil, "init", "-q", "--bare")

	t.Run("fetch branch tips", func(t *testing.T) {
		require.NoError(t, s.FetchObjects(ctx, fx.source, []sha.SHA{fx.main, fx.clean}))

		assert.Equal(t, "commit", gtxGit(ctx, t, s.Directory(), nil, "cat-file", "-t", fx.main.String()))
		assert.Equal(t, "commit", gtxGit(ctx, t, s.Directory(), nil, "cat-file", "-t", fx.clean.String()))
	})

	t.Run("unknown source", func(t *testing.T) {
		unknown := sha.Must("1234567890123456789012345678901234567890")
		err := s.FetchObjects(ctx, filepath.Join(t.TempDir(), "gone"), []sha.SHA{unknown})
		require.Error(t, err)
	})

	t.Run("unknown object", func(t *testing.T) {
		unknown := sha.Must("1234567890123456789012345678901234567890")
		err := s.FetchObjects(ctx, fx.source, []sha.SHA{unknown})
		require.Error(t, err)
	})
}

func TestGtxProvideRunnerFactory(t *testing.T) {
	root := t.TempDir()

	factory, err := ProvideRunnerFactory(types.Config{Root: root})
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, repoSharedRepoSubdirName), factory.Directory())
	assert.DirExists(t, factory.Directory())

	file := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(file, nil, 0o600))
	_, err = ProvideRunnerFactory(types.Config{Root: file})
	require.Error(t, err)
}
