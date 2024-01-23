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

package adapter_test

import (
	"context"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/git/adapter"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/git/types"

	gitea "code.gitea.io/gitea/modules/git"
)

type teardown func()

var (
	testAuthor = &gitea.Signature{
		Name:  "test",
		Email: "test@test.com",
	}

	testCommitter = &gitea.Signature{
		Name:  "test",
		Email: "test@test.com",
	}
)

type mockClientFactory struct{}

func (f *mockClientFactory) NewClient(_ context.Context, _ map[string]string) (hook.Client, error) {
	return hook.NewNoopClient([]string{"mocked client"}), nil
}

func setupGit(t *testing.T) adapter.Adapter {
	t.Helper()
	git, err := adapter.New(
		types.Config{Trace: true},
		adapter.NewInMemoryLastCommitCache(5*time.Minute),
		&mockClientFactory{},
	)
	if err != nil {
		t.Fatalf("error initializing repository: %v", err)
	}
	return git
}

func setupRepo(t *testing.T, git adapter.Adapter, name string) (*gitea.Repository, teardown) {
	t.Helper()
	ctx := context.Background()

	tmpdir := os.TempDir()

	repoPath := path.Join(tmpdir, "test_repos", name)

	err := git.InitRepository(ctx, repoPath, true)
	if err != nil {
		t.Fatalf("error initializing repository: %v", err)
	}

	repo, err := git.OpenRepository(ctx, repoPath)
	if err != nil {
		t.Fatalf("error opening repository '%s': %v", name, err)
	}

	err = repo.SetDefaultBranch("main")
	if err != nil {
		t.Fatalf("error setting default branch 'main': %v", err)
	}

	err = git.Config(ctx, repoPath, "user.email", testCommitter.Email)
	if err != nil {
		t.Fatalf("error setting config user.email %s: %v", testCommitter.Email, err)
	}

	err = git.Config(ctx, repoPath, "user.name", testCommitter.Name)
	if err != nil {
		t.Fatalf("error setting config user.name %s: %v", testCommitter.Name, err)
	}

	return repo, func() {
		if err := os.RemoveAll(repoPath); err != nil {
			t.Errorf("error while removeng the repository '%s'", repoPath)
		}
	}
}

func writeFile(
	t *testing.T,
	repo *gitea.Repository,
	path string,
	content string,
	parents []string,
) (oid gitea.SHA1, commitSha gitea.SHA1) {
	t.Helper()
	oid, err := repo.HashObject(strings.NewReader(content))
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
