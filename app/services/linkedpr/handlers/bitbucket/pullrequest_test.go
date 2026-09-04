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

package bitbucket

import (
	"testing"

	"github.com/harness/gitness/app/services/linkedpr"

	"github.com/stretchr/testify/require"
)

func TestPRSyncSpec(t *testing.T) {
	entries := PRSyncSpec(linkedpr.PullRequestPayload{
		Number: 8, HeadRef: "feature", BaseRef: "main",
	})

	require.Len(t, entries, 2)
	require.Equal(t, "refs/heads/main", entries[0].RemoteRef)
	require.Empty(t, entries[0].LocalRef)
	// The source branch is the only head Bitbucket Cloud offers, so it is the
	// ref that gets renamed into the Harness Code PR namespace.
	require.Equal(t, "refs/heads/feature", entries[1].RemoteRef)
	require.Equal(t, "refs/pullreq/8/head", entries[1].LocalRef)
}
