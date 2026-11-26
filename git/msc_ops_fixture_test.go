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
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/git/sharedrepo"
	"github.com/harness/gitness/git/types"

	"github.com/stretchr/testify/require"
)

const mscTestGitHookPath = "/path/to/githook"

type mscNoopHookClientFactory struct{}

func (mscNoopHookClientFactory) NewClient(map[string]string) (hook.Client, error) {
	return mscNoopHookClient{}, nil
}

type mscNoopHookClient struct{}

func (mscNoopHookClient) PreReceive(context.Context, hook.PreReceiveInput) (hook.Output, error) {
	return hook.Output{}, nil
}

func (mscNoopHookClient) Update(context.Context, hook.UpdateInput) (hook.Output, error) {
	return hook.Output{}, nil
}

func (mscNoopHookClient) PostReceive(context.Context, hook.PostReceiveInput) (hook.Output, error) {
	return hook.Output{}, nil
}

// mscOpsFixture is a service with one repository: main has a.txt, dir/b.txt and dir/sub/c.txt.
type mscOpsFixture struct {
	svc      *Service
	repoUID  string
	repoPath string
	actor    Identity
}

func mscNewOpsFixture(t *testing.T) *mscOpsFixture {
	t.Helper()
	requireGit(t)

	config := types.Config{Root: t.TempDir(), HookPath: mscTestGitHookPath}

	adapter, err := api.New(config, api.NewInMemoryLastCommitCache(time.Minute), nil)
	require.NoError(t, err)
	runnerFactory, err := sharedrepo.NewRunnerFactory(config.Root)
	require.NoError(t, err)
	svc, err := New(config, adapter, mscNoopHookClientFactory{}, nil, runnerFactory)
	require.NoError(t, err)

	actor := Identity{Name: "MscActor", Email: "msc-actor@example.com"}
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

	return &mscOpsFixture{
		svc:      svc,
		repoUID:  out.UID,
		repoPath: getFullPathForRepo(svc.reposRoot, out.UID),
		actor:    actor,
	}
}

func (f *mscOpsFixture) mscRead() ReadParams { return ReadParams{RepoUID: f.repoUID} }

func (f *mscOpsFixture) mscWrite() WriteParams {
	return WriteParams{RepoUID: f.repoUID, Actor: f.actor}
}

func (f *mscOpsFixture) mscHead(ctx context.Context, t *testing.T) string {
	t.Helper()
	full := []string{
		"-c", "user.name=Fixture", "-c", "user.email=fixture@example.com",
		"-c", "maintenance.auto=false", "-c", "gc.auto=0",
		"rev-parse", "main",
	}
	cmd := exec.CommandContext(ctx, "git", full...)
	cmd.Dir = f.repoPath
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git rev-parse main: %s", out)
	return strings.TrimSpace(string(out))
}
