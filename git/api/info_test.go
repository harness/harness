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
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadRepositoryInfo_RealRepo(t *testing.T) {
	requireGit(t)
	g := &Git{}
	repo := initRepoWithCommit(t)
	runGit(t, repo, "config", "maintenance.auto", "false")
	runGit(t, repo, "config", "gc.auto", "0")
	commitFile(t, repo, "b.txt", "b\n", "second")
	gitDir := filepath.Join(repo, ".git")

	info, err := LoadRepositoryInfo(gitDir)
	require.NoError(t, err)
	require.Positive(t, info.LooseObjects.Count)
	require.Positive(t, info.LooseObjects.Size)
	require.Zero(t, info.LooseObjects.StaleCount)
	require.Zero(t, info.PackFiles.Count)
	require.Positive(t, info.References.LooseReferenceCount)
	require.False(t, info.CommitGraph.Exists)

	require.NoError(t, g.RepackObjects(t.Context(), repo, RepackParams{
		SinglePack: true, RemoveRedundantObjects: true, WriteBitmap: true,
	}))
	require.NoError(t, g.CommitGraph(t.Context(), repo, CommitGraphParams{
		Action: CommitGraphActionWrite, Reachable: true, ChangedPaths: true,
	}))
	runGit(t, repo, "pack-refs", "--all")

	info, err = LoadRepositoryInfo(gitDir)
	require.NoError(t, err)
	require.Equal(t, uint64(1), info.PackFiles.Count)
	require.Positive(t, info.PackFiles.Size)
	require.Equal(t, uint8(1), uint8(info.PackFiles.Bitmap.Version))
	require.Positive(t, info.References.PackedReferenceSize)
	require.True(t, info.CommitGraph.Exists)
	require.Zero(t, info.CommitGraph.ChainLength)
	require.True(t, info.CommitGraph.HasBloomFilters)

	// split commit graph produces a chain
	require.NoError(t, g.CommitGraph(t.Context(), repo, CommitGraphParams{
		Action: CommitGraphActionWrite, Reachable: true, Split: new(CommitGraphSplitOption),
	}))
	info, err = LoadRepositoryInfo(gitDir)
	require.NoError(t, err)
	require.True(t, info.CommitGraph.Exists)
}

func TestLoadRepositoryInfo_MissingDir(t *testing.T) {
	_, err := LoadRepositoryInfo(filepath.Join(t.TempDir(), "missing"))
	require.Error(t, err)
}

func TestLoadLooseObjectsInfo(t *testing.T) {
	dir := t.TempDir()
	objects := filepath.Join(dir, "objects")
	require.NoError(t, os.MkdirAll(filepath.Join(objects, "ab"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(objects, "notashard"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(objects, "pack"), 0o755))

	fresh := filepath.Join(objects, "ab", "cdef0123")
	stale := filepath.Join(objects, "ab", "0123abcd")
	garbage := filepath.Join(objects, "ab", "tmp_obj_123")
	require.NoError(t, os.WriteFile(fresh, []byte("12345"), 0o600))
	require.NoError(t, os.WriteFile(stale, []byte("123"), 0o600))
	require.NoError(t, os.WriteFile(garbage, []byte("12"), 0o600))
	old := time.Now().Add(-30 * 24 * time.Hour)
	require.NoError(t, os.Chtimes(stale, old, old))

	info, err := LoadLooseObjectsInfo(dir, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.Equal(t, LooseObjectsInfo{
		Count: 2, Size: 8, StaleCount: 1, StaleSize: 3, GarbageCount: 1, GarbageSize: 2,
	}, info)

	_, err = LoadLooseObjectsInfo(filepath.Join(dir, "nope"), time.Now())
	require.Error(t, err)
}

func TestLoadPackFilesInfo(t *testing.T) {
	t.Run("no pack dir", func(t *testing.T) {
		info, err := LoadPackFilesInfo(t.TempDir())
		require.NoError(t, err)
		require.Equal(t, PackFilesInfo{}, info)
	})

	t.Run("classifies files", func(t *testing.T) {
		dir := t.TempDir()
		packs := filepath.Join(dir, "objects", "pack")
		require.NoError(t, os.MkdirAll(packs, 0o755))

		write := func(name string, data []byte) {
			require.NoError(t, os.WriteFile(filepath.Join(packs, name), data, 0o600))
		}
		write("pack-1.pack", []byte("1234"))
		write("pack-1.keep", []byte("12"))
		write("pack-1.mtimes", []byte("123"))
		write("pack-1.rev", []byte("1"))
		write("pack-1.bitmap", bitmapHeader(1, 0x04|0x10))
		write("multi-pack-index-abc.bitmap", bitmapHeader(1, 0))
		write("multi-pack-index", midxHeader(1, 0, 3))
		write("junk.txt", []byte("12345"))

		repackTime := time.Unix(1700000000, 0)
		require.NoError(t, SetLastFullRepackTime(dir, repackTime))

		info, err := LoadPackFilesInfo(dir)
		require.NoError(t, err)
		require.Equal(t, uint64(1), info.Count)
		require.Equal(t, uint64(4), info.Size)
		require.Equal(t, uint64(1), info.KeepCount)
		require.Equal(t, uint64(2), info.KeepSize)
		require.Equal(t, uint64(1), info.CruftCount)
		require.Equal(t, uint64(3), info.CruftSize)
		require.Equal(t, uint64(1), info.ReverseIndexCount)
		require.Equal(t, uint64(1), info.GarbageCount)
		require.Equal(t, uint64(5), info.GarbageSize)
		require.Equal(t, BitmapInfo{Version: 1, HasHashCache: true, HasLookupTable: true}, info.Bitmap)
		require.Equal(t, BitmapInfo{Version: 1}, info.MultiPackIndexBitmap)
		require.Equal(t, MultiPackIndexInfo{Exists: true, Version: 1, PackFileCount: 3}, info.MultiPackIndex)
		require.True(t, info.LastFullRepack.Equal(repackTime))
	})

	t.Run("invalid bitmap", func(t *testing.T) {
		dir := t.TempDir()
		packs := filepath.Join(dir, "objects", "pack")
		require.NoError(t, os.MkdirAll(packs, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(packs, "pack-1.bitmap"), []byte("bad"), 0o600))
		_, err := LoadPackFilesInfo(dir)
		require.Error(t, err)
	})

	t.Run("invalid multi pack index", func(t *testing.T) {
		dir := t.TempDir()
		packs := filepath.Join(dir, "objects", "pack")
		require.NoError(t, os.MkdirAll(packs, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(packs, "multi-pack-index"), []byte("bad"), 0o600))
		_, err := LoadPackFilesInfo(dir)
		require.Error(t, err)
	})
}

func TestFullRepackTime(t *testing.T) {
	dir := t.TempDir()

	got, err := GetLastFullRepackTime(dir)
	require.NoError(t, err)
	require.True(t, got.IsZero())

	first := time.Unix(1600000000, 0)
	require.NoError(t, SetLastFullRepackTime(dir, first))
	got, err = GetLastFullRepackTime(dir)
	require.NoError(t, err)
	require.True(t, got.Equal(first))

	// updating an existing marker file
	second := time.Unix(1700000000, 0)
	require.NoError(t, SetLastFullRepackTime(dir, second))
	got, err = GetLastFullRepackTime(dir)
	require.NoError(t, err)
	require.True(t, got.Equal(second))

	require.Error(t, SetLastFullRepackTime(filepath.Join(dir, "missing"), second))
}

func TestLoadBitmapInfo_Errors(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, data []byte) string {
		p := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(p, data, 0o600))
		return p
	}

	_, err := LoadBitmapInfo(filepath.Join(dir, "missing"))
	require.Error(t, err)

	_, err = LoadBitmapInfo(write("short", []byte("BI")))
	require.Error(t, err)

	_, err = LoadBitmapInfo(write("sig", []byte("XXXX\x00\x01\x00\x00")))
	require.Error(t, err)

	_, err = LoadBitmapInfo(write("version", bitmapHeader(2, 0)))
	require.Error(t, err)
}

func TestLoadMultiPackIndexInfo_Errors(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, data []byte) string {
		p := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(p, data, 0o600))
		return p
	}

	_, err := LoadMultiPackIndexInfo(filepath.Join(dir, "missing"))
	require.Error(t, err)

	_, err = LoadMultiPackIndexInfo(write("short", []byte("MID")))
	require.Error(t, err)

	_, err = LoadMultiPackIndexInfo(write("sig", append([]byte("XXXX"), make([]byte, 8)...)))
	require.Error(t, err)

	_, err = LoadMultiPackIndexInfo(write("version", midxHeader(2, 0, 1)))
	require.Error(t, err)

	_, err = LoadMultiPackIndexInfo(write("base", midxHeader(1, 1, 1)))
	require.Error(t, err)
}

func TestLoadReferencesInfo(t *testing.T) {
	dir := t.TempDir()

	info, err := LoadReferencesInfo(dir)
	require.NoError(t, err)
	require.Equal(t, ReferencesInfo{}, info)

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "refs", "heads"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "refs", "heads", "a"), []byte("x"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "refs", "heads", "b"), []byte("x"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "packed-refs"), []byte("12345"), 0o600))

	info, err = LoadReferencesInfo(dir)
	require.NoError(t, err)
	require.Equal(t, ReferencesInfo{LooseReferenceCount: 2, PackedReferenceSize: 5}, info)
}

func TestLoadCommitGraphInfo_Synthetic(t *testing.T) {
	t.Run("none", func(t *testing.T) {
		info, err := LoadCommitGraphInfo(t.TempDir())
		require.NoError(t, err)
		require.False(t, info.Exists)
	})

	t.Run("chain with all chunk types", func(t *testing.T) {
		dir := t.TempDir()
		graphs := filepath.Join(dir, "objects", "info", "commit-graphs")
		require.NoError(t, os.MkdirAll(graphs, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(graphs, "commit-graph-chain"), []byte("aaa\nbbb\n"), 0o600))
		// first graph exists with bloom + gda2 + gdo2 chunks, the second one vanished concurrently
		require.NoError(t, os.WriteFile(filepath.Join(graphs, "graph-aaa.graph"),
			commitGraphFile([]string{"BIDX", "BDAT", "GDA2", "GDO2"}), 0o600))

		info, err := LoadCommitGraphInfo(dir)
		require.NoError(t, err)
		require.Equal(t, CommitGraphInfo{
			Exists: true, ChainLength: 2, HasBloomFilters: true,
			HasGenerationData: true, HasGenerationDataOverflow: true,
		}, info)
	})

	t.Run("invalid graph files", func(t *testing.T) {
		for name, data := range map[string][]byte{
			"empty":     {},
			"signature": []byte("XXXX\x01\x01\x00\x00"),
			"version":   []byte("CGPH\x02\x01\x00\x00"),
			"table":     []byte("CGPH\x01\x01\x05\x00"),
		} {
			t.Run(name, func(t *testing.T) {
				dir := t.TempDir()
				infoDir := filepath.Join(dir, "objects", "info")
				require.NoError(t, os.MkdirAll(infoDir, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(infoDir, "commit-graph"), data, 0o600))
				_, err := LoadCommitGraphInfo(dir)
				require.Error(t, err)
			})
		}
	})
}

func bitmapHeader(version uint16, flags uint16) []byte {
	h := []byte{'B', 'I', 'T', 'M', 0, 0, 0, 0}
	binary.BigEndian.PutUint16(h[4:6], version)
	binary.BigEndian.PutUint16(h[6:8], flags)
	return h
}

func midxHeader(version, baseFiles uint8, packs uint32) []byte {
	h := []byte{'M', 'I', 'D', 'X', version, 1, 0, baseFiles, 0, 0, 0, 0}
	binary.BigEndian.PutUint32(h[8:12], packs)
	return h
}

// commitGraphFile builds a minimal commit-graph file whose chunk table lists the given chunk ids.
func commitGraphFile(chunks []string) []byte {
	data := []byte{'C', 'G', 'P', 'H', 1, 1, byte(len(chunks)), 0}
	for _, c := range chunks {
		entry := make([]byte, 12)
		copy(entry, c)
		data = append(data, entry...)
	}
	return append(data, make([]byte, 12)...)
}
