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

package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMaintenanceCommands(t *testing.T) {
	requireGit(t)
	g := &Git{}
	repo := initRepoWithCommit(t)
	commitFile(t, repo, "b.txt", "b\n", "second")
	truth := true
	falseVal := false

	t.Run("gc", func(t *testing.T) {
		require.NoError(t, g.GC(t.Context(), repo, GCParams{}))
		require.NoError(t, g.GC(t.Context(), repo, GCParams{Auto: true, Cruft: &truth}))
		require.NoError(t, g.GC(t.Context(), repo, GCParams{Cruft: &falseVal, Prune: false}))
		require.NoError(t, g.GC(t.Context(), repo, GCParams{Prune: "now"}))
		require.NoError(t, g.GC(t.Context(), repo, GCParams{Prune: time.Now().Add(-time.Hour)}))
		require.NoError(t, g.GC(t.Context(), repo, GCParams{Prune: time.Time{}}))
		require.Error(t, g.GC(t.Context(), repo, GCParams{Prune: 12}))
		require.Error(t, g.GC(t.Context(), t.TempDir(), GCParams{}))
	})

	t.Run("repack", func(t *testing.T) {
		require.NoError(t, g.RepackObjects(t.Context(), repo, RepackParams{
			SinglePack: true, RemoveRedundantObjects: true, Local: true,
		}))
		require.NoError(t, g.RepackObjects(t.Context(), repo, RepackParams{
			SinglePack: true, KeepUnreachable: true, WriteBitmap: true,
		}))
		require.NoError(t, g.RepackObjects(t.Context(), repo, RepackParams{
			SinglePack: true, RemoveRedundantObjects: true, Cruft: true,
			CruftExpireBefore: time.Now().Add(-time.Hour), PackKeptObjects: true,
		}))
		require.Error(t, g.RepackObjects(t.Context(), t.TempDir(), RepackParams{}))
	})

	t.Run("prune packed", func(t *testing.T) {
		require.NoError(t, g.PrunePacked(t.Context(), repo, PrunePackedParams{DryRun: true, Quiet: true}))
		require.Error(t, g.PrunePacked(t.Context(), t.TempDir(), PrunePackedParams{}))
	})

	t.Run("prune objects", func(t *testing.T) {
		require.NoError(t, g.PruneObjects(t.Context(), repo, PruneObjectsParams{
			DryRun: true, ExpireBefore: time.Now().Add(-time.Hour),
		}))
		require.NoError(t, g.PruneObjects(t.Context(), repo, PruneObjectsParams{}))
		require.Error(t, g.PruneObjects(t.Context(), t.TempDir(), PruneObjectsParams{}))
	})
}

func TestCommitGraph(t *testing.T) {
	requireGit(t)
	g := &Git{}
	repo := initRepoWithCommit(t)
	commitFile(t, repo, "b.txt", "b\n", "second")

	empty := CommitGraphSplitOptionEmpty
	replace := CommitGraphSplitOptionReplace
	bad := CommitGraphSplitOption("bad")

	t.Run("validation", func(t *testing.T) {
		require.NoError(t, CommitGraphActionWrite.Validate())
		require.NoError(t, CommitGraphActionVerify.Validate())
		require.Error(t, CommitGraphAction("x").Validate())

		s, err := CommitGraphActionWrite.String()
		require.NoError(t, err)
		require.Equal(t, "write", s)
		s, err = CommitGraphActionVerify.String()
		require.NoError(t, err)
		require.Equal(t, "verify", s)
		_, err = CommitGraphAction("x").String()
		require.Error(t, err)

		require.NoError(t, empty.Validate())
		require.NoError(t, replace.Validate())
		require.Error(t, bad.Validate())

		require.Error(t, CommitGraphParams{Action: "x"}.Validate())
		require.Error(t, CommitGraphParams{Action: CommitGraphActionWrite, Split: &bad}.Validate())
		require.NoError(t, CommitGraphParams{Action: CommitGraphActionWrite, Split: &replace}.Validate())
	})

	t.Run("write and verify", func(t *testing.T) {
		require.NoError(t, g.CommitGraph(t.Context(), repo, CommitGraphParams{
			Action: CommitGraphActionWrite, Reachable: true, ChangedPaths: true,
		}))
		require.NoError(t, g.CommitGraph(t.Context(), repo, CommitGraphParams{
			Action: CommitGraphActionWrite, Reachable: true, Split: &empty, SizeMultiple: 2,
		}))
		require.NoError(t, g.CommitGraph(t.Context(), repo, CommitGraphParams{
			Action: CommitGraphActionWrite, Reachable: true, Split: &replace,
		}))
		require.NoError(t, g.CommitGraph(t.Context(), repo, CommitGraphParams{Action: CommitGraphActionVerify}))
	})

	t.Run("invalid params", func(t *testing.T) {
		require.Error(t, g.CommitGraph(t.Context(), repo, CommitGraphParams{Action: "x"}))
	})
}
