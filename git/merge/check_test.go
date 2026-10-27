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

package merge

import (
	"testing"

	"github.com/harness/gitness/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommitCount_UnknownRevision(t *testing.T) {
	requireGit(t)

	repoPath := newTestRepo(t)

	// empty repository: the branch doesn't resolve yet.
	_, err := CommitCount(t.Context(), repoPath, "", "refs/heads/main")
	require.Error(t, err)
	assert.True(t, errors.IsNotFound(err), "expected not found, got: %v", err)

	runGit(t, repoPath, "commit", "--allow-empty", "-m", "first")

	count, err := CommitCount(t.Context(), repoPath, "", "refs/heads/main")
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	_, err = CommitCount(t.Context(), repoPath, "", "refs/heads/gone")
	require.Error(t, err)
	assert.True(t, errors.IsNotFound(err), "expected not found, got: %v", err)
}
