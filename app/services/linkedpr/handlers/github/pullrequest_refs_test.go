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

package github

import (
	"fmt"
	"testing"

	"github.com/harness/gitness/app/services/linkedpr"

	"github.com/stretchr/testify/require"
)

func TestGithubPRSyncSpec(t *testing.T) {
	t.Parallel()

	payload := linkedpr.PullRequestPayload{
		Number:  8,
		HeadRef: "feature-branch",
		BaseRef: "main",
	}

	entries := githubPRSyncSpec(payload)

	require.Len(t, entries, 3)

	// Base branch is fetched verbatim (empty LocalRef).
	require.Equal(t, "refs/heads/main", entries[0].RemoteRef)
	require.Empty(t, entries[0].LocalRef, "verbatim entry must have empty LocalRef")

	// Head branch is fetched verbatim so its objects are available locally.
	require.Equal(t, "refs/heads/feature-branch", entries[1].RemoteRef)
	require.Empty(t, entries[1].LocalRef, "verbatim entry must have empty LocalRef")

	// PR head is renamed: refs/pull/<N>/head → refs/pullreq/<N>/head.
	// The "pull" → "pullreq" rename is the core of this feature — assert both sides.
	require.Equal(t, fmt.Sprintf("refs/pull/%d/head", payload.Number), entries[2].RemoteRef)
	require.Equal(t, fmt.Sprintf("refs/pullreq/%d/head", payload.Number), entries[2].LocalRef)
	require.NotEqual(t, entries[2].RemoteRef, entries[2].LocalRef,
		"remote and local refs must differ (pull vs pullreq namespace)")
}
