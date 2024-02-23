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

package api_test

import (
	"context"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/git/types"
)

type teardown func()

var (
	testAuthor = &api.Signature{
		Identity: api.Identity{
			Name:  "test",
			Email: "test@test.com",
		},
	}

	testCommitter = &api.Signature{
		Identity: api.Identity{
			Name:  "test",
			Email: "test@test.com",
		},
	}
)

type mockClientFactory struct{}

func (f *mockClientFactory) NewClient(_ context.Context, _ map[string]string) (hook.Client, error) {
	return hook.NewNoopClient([]string{"mocked client"}), nil
}

func setupGit(t *testing.T) *api.Git {
	t.Helper()
	git, err := api.New(
		types.Config{Trace: true},
		api.NewInMemoryLastCommitCache(5*time.Minute),
		&mockClientFactory{},
	)
	if err != nil {
		t.Fatalf("error initializing repository: %v", err)
	}
	return git
}

func setupRepo(t *testing.T, git *api.Git, name string) teardown {
	t.Helper()
	ctx := context.Background()

	tmpdir := os.TempDir()

	repoPath := path.Join(tmpdir, "test_repos", name)

	err := git.InitRepository(ctx, repoPath, true)
	if err != nil {
		t.Fatalf("error initializing repository: %v", err)
	}

	err = git.SetDefaultBranch(ctx, repoPath, "main", true)
	if err != nil {
		t.Fatalf("error setting default branch 'main': %v", err)
	}

	err = git.Config(ctx, repoPath, "user.email", testCommitter.Identity.Email)
	if err != nil {
		t.Fatalf("error setting config user.email %s: %v", testCommitter.Identity.Email, err)
	}

	err = git.Config(ctx, repoPath, "user.name", testCommitter.Identity.Name)
	if err != nil {
		t.Fatalf("error setting config user.name %s: %v", testCommitter.Identity.Name, err)
	}

	return func() {
		if err := os.RemoveAll(repoPath); err != nil {
			t.Errorf("error while removeng the repository '%s'", repoPath)
		}
	}
}

func writeFile(
	t *testing.T,
	git *api.Git,
	repoPath string,
	path string,
	content string,
	parents []string,
) (oid api.SHA, commitSha api.SHA) {
	t.Helper()
	oid, err := git.HashObject(context.Background(), repoPath, strings.NewReader(content))
	if err != nil {
		t.Fatalf("failed to hash object: %v", err)
	}

	err = repo.AddObjectToIndex("100644", oid, path)
	if err != nil {
		t.Fatalf("failed to add object to index: %v", err)
	}

	tree, err := repo.WriteTree()
	if err != nil {
		t.Fatalf("failed to write tree: %v", err)
	}

	commitSha, err = repo.CommitTree(testAuthor, testCommitter, tree, gitea.CommitTreeOpts{
		Message: "write file operation",
		Parents: parents,
	})
	if err != nil {
		t.Fatalf("failed to commit tree: %v", err)
	}
	return oid, commitSha
}
