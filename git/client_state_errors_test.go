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
	"os/exec"
	"testing"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testRepoUID = "abcdef0123456789"

// newBareTestService creates a service that is backed by an empty bare repository.
func newBareTestService(t *testing.T) *Service {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available on PATH")
	}

	root := t.TempDir()
	repoPath := getFullPathForRepo(root, testRepoUID)

	out, err := exec.CommandContext(t.Context(), "git", "init", "--bare", "--initial-branch=main", repoPath).
		CombinedOutput()
	require.NoErrorf(t, err, "git init failed: %s", out)

	return &Service{reposRoot: root, git: &api.Git{}}
}

// TestSummary_EmptyRepo asserts that an empty repository has an all-zero summary.
func TestSummary_EmptyRepo(t *testing.T) {
	svc := newBareTestService(t)

	out, err := svc.Summary(t.Context(), SummaryParams{ReadParams: ReadParams{RepoUID: testRepoUID}})
	require.NoError(t, err)
	assert.Equal(t, 0, out.CommitCount)
	assert.Equal(t, 0, out.BranchCount)
	assert.Equal(t, 0, out.TagCount)
}

// TestServicePack_NotOurRef asserts that wanting a missing object returns InvalidArgument.
func TestServicePack_NotOurRef(t *testing.T) {
	svc := newBareTestService(t)

	const missingSHA = "1111111111111111111111111111111111111111"

	// pkt-line "want <sha>\n" (0x32 bytes), flush, pkt-line "done\n".
	stdin := bytes.NewBufferString("0032want " + missingSHA + "\n00000009done\n")

	err := svc.ServicePack(t.Context(), &ServicePackParams{
		ReadParams: &ReadParams{RepoUID: testRepoUID},
		ServicePackOptions: api.ServicePackOptions{
			Service:      enum.GitServiceTypeUploadPack,
			StatelessRPC: true,
			Stdin:        stdin,
			Stdout:       &bytes.Buffer{},
		},
	})
	require.Error(t, err)
	assert.True(t, errors.IsInvalidArgument(err), "expected invalid argument, got: %v", err)
	assert.Contains(t, err.Error(), missingSHA)
}
