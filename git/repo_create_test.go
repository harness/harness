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
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/git/sharedrepo"
	"github.com/harness/gitness/git/types"

	"github.com/stretchr/testify/require"
)

const testGitHookPath = "/path/to/githook"

func TestCreateRepository(t *testing.T) {
	requireGit(t)

	actor := Identity{Name: "Actor", Email: "actor@example.com"}
	author := Identity{Name: "Author", Email: "author@example.com"}
	date := time.Date(2024, 3, 4, 5, 6, 7, 0, time.UTC)

	t.Run("with files", func(t *testing.T) {
		s, root := newTestService(t)

		out, err := s.CreateRepository(context.Background(), &CreateRepositoryParams{
			Actor:         actor,
			DefaultBranch: "trunk",
			Files: []File{
				{Path: "README.md", Content: []byte("# readme\n")},
				{Path: "/docs/../docs/a/b.txt", Content: []byte("b")},
				{Path: "empty", Content: nil},
			},
			Author:        &author,
			AuthorDate:    &date,
			CommitterDate: &date,
		})
		require.NoError(t, err)

		repoPath := getFullPathForRepo(root, out.UID)

		require.Equal(t, "refs/heads/trunk", runGit(t, repoPath, "symbolic-ref", "HEAD"))
		require.Equal(t, "refs/heads/trunk", runGit(t, repoPath, "for-each-ref", "--format=%(refname)"))
		require.Equal(t,
			"Author|author@example.com|Actor|actor@example.com|initial commit|",
			runGit(t, repoPath, "log", "-1", "--format=%an|%ae|%cn|%ce|%s|%P", "trunk"))
		require.Equal(t, strconv.FormatInt(date.Unix(), 10), runGit(t, repoPath, "log", "-1", "--format=%at", "trunk"))
		require.Equal(t, strconv.FormatInt(date.Unix(), 10), runGit(t, repoPath, "log", "-1", "--format=%ct", "trunk"))

		require.Equal(t, "README.md\ndocs/a/b.txt\nempty",
			runGit(t, repoPath, "ls-tree", "-r", "--name-only", "trunk"))
		require.Equal(t, "# readme", runGit(t, repoPath, "cat-file", "-p", "trunk:README.md"))
		require.Equal(t, "b", runGit(t, repoPath, "cat-file", "-p", "trunk:docs/a/b.txt"))
		require.Equal(t, "0", runGit(t, repoPath, "cat-file", "-s", "trunk:empty"))

		requireHooksInstalled(t, repoPath)
		requireNoSharedRepoLeftovers(t, root)
	})

	t.Run("committer and author default to actor", func(t *testing.T) {
		s, root := newTestService(t)

		out, err := s.CreateRepository(context.Background(), &CreateRepositoryParams{
			Actor:         actor,
			DefaultBranch: "main",
			Files:         []File{{Path: "f", Content: []byte("x")}},
		})
		require.NoError(t, err)

		repoPath := getFullPathForRepo(root, out.UID)
		require.Equal(t, "Actor|actor@example.com|Actor|actor@example.com",
			runGit(t, repoPath, "log", "-1", "--format=%an|%ae|%cn|%ce", "main"))
	})

	t.Run("without files", func(t *testing.T) {
		s, root := newTestService(t)

		out, err := s.CreateRepository(context.Background(), &CreateRepositoryParams{
			Actor:         actor,
			DefaultBranch: "main",
		})
		require.NoError(t, err)

		repoPath := getFullPathForRepo(root, out.UID)

		require.Equal(t, "refs/heads/main", runGit(t, repoPath, "symbolic-ref", "HEAD"))
		require.Empty(t, runGit(t, repoPath, "for-each-ref"))

		requireHooksInstalled(t, repoPath)
		requireNoSharedRepoLeftovers(t, root)
	})

	t.Run("invalid path removes the repository", func(t *testing.T) {
		s, root := newTestService(t)

		const repoUID = "abcdef0123456789"

		_, err := s.CreateRepository(context.Background(), &CreateRepositoryParams{
			RepoUID:       repoUID,
			Actor:         actor,
			DefaultBranch: "main",
			Files: []File{
				{Path: "ok.txt", Content: []byte("ok")},
				{Path: ".git/config", Content: []byte("nope")},
			},
		})
		require.Error(t, err)

		require.NoDirExists(t, getFullPathForRepo(root, repoUID))
		requireNoSharedRepoLeftovers(t, root)
	})

	t.Run("existing repository conflicts", func(t *testing.T) {
		s, _ := newTestService(t)

		params := &CreateRepositoryParams{
			RepoUID:       "abcdef0123456789",
			Actor:         actor,
			DefaultBranch: "main",
		}

		_, err := s.CreateRepository(context.Background(), params)
		require.NoError(t, err)

		_, err = s.CreateRepository(context.Background(), params)
		require.Error(t, err)
	})
}

func newTestService(t *testing.T) (*Service, string) {
	t.Helper()

	config := types.Config{
		Root:     t.TempDir(),
		HookPath: testGitHookPath,
	}

	adapter, err := api.New(config, nil, nil)
	require.NoError(t, err)

	runnerFactory, err := sharedrepo.NewRunnerFactory(config.Root)
	require.NoError(t, err)

	s, err := New(config, adapter, failingHookClientFactory{t: t}, nil, runnerFactory)
	require.NoError(t, err)

	return s, s.reposRoot
}

// failingHookClientFactory creates hook clients that fail the test if any hook is called.
type failingHookClientFactory struct {
	t *testing.T
}

func (f failingHookClientFactory) NewClient(map[string]string) (hook.Client, error) {
	return failingHookClient(f), nil
}

type failingHookClient struct {
	t *testing.T
}

func (c failingHookClient) PreReceive(context.Context, hook.PreReceiveInput) (hook.Output, error) {
	c.t.Error("unexpected call of the pre-receive hook")
	return hook.Output{}, nil
}

func (c failingHookClient) Update(context.Context, hook.UpdateInput) (hook.Output, error) {
	c.t.Error("unexpected call of the update hook")
	return hook.Output{}, nil
}

func (c failingHookClient) PostReceive(context.Context, hook.PostReceiveInput) (hook.Output, error) {
	c.t.Error("unexpected call of the post-receive hook")
	return hook.Output{}, nil
}

func requireGit(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available on PATH")
	}
}

func runGit(t *testing.T, repoPath string, args ...string) string {
	t.Helper()

	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = repoPath

	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %s: %s", strings.Join(args, " "), out)

	return strings.TrimSpace(string(out))
}

func requireHooksInstalled(t *testing.T, repoPath string) {
	t.Helper()

	for _, hookName := range gitServerHookNames {
		target, err := os.Readlink(filepath.Join(repoPath, gitHooksDir, hookName))
		require.NoError(t, err)
		require.Equal(t, testGitHookPath, target)
	}
}

func requireNoSharedRepoLeftovers(t *testing.T, root string) {
	t.Helper()

	entries, err := os.ReadDir(filepath.Join(filepath.Dir(root), "shared_temp"))
	if os.IsNotExist(err) {
		return
	}
	require.NoError(t, err)
	require.Empty(t, entries)
}
