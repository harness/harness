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
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	// StaleObjectsGracePeriod is time delta that is used to indicate cutoff wherein an object
	// would be considered old.
	StaleObjectsGracePeriod = -7 * 24 * time.Hour
	// FullRepackTimestampFileName is the name of the file last full repack.
	FullRepackTimestampFileName = ".harness-full-repack-timestamp"
)

// RepositoryInfo contains information about the repository.
type RepositoryInfo struct {
	LooseObjects LooseObjectsInfo
	PackFiles    PackFilesInfo
	References   ReferencesInfo
	CommitGraph  CommitGraphInfo
}

// LoadRepositoryInfo tracks all git files and returns the stats of repository.
func LoadRepositoryInfo(repoPath string) (RepositoryInfo, error) {
	var result RepositoryInfo
	var err error

	result.LooseObjects, err = LoadLooseObjectsInfo(repoPath, time.Now().Add(StaleObjectsGracePeriod))
	if err != nil {
		return RepositoryInfo{}, fmt.Errorf("loading loose objects info: %w", err)
	}

	result.PackFiles, err = LoadPackFilesInfo(repoPath)
	if err != nil {
		return RepositoryInfo{}, fmt.Errorf("loading pack files info: %w", err)
	}

	result.References, err = LoadReferencesInfo(repoPath)
	if err != nil {
		return RepositoryInfo{}, fmt.Errorf("loading references info: %w", err)
	}

	result.CommitGraph, err = LoadCommitGraphInfo(repoPath)
	if err != nil {
		return RepositoryInfo{}, fmt.Errorf("loading commit graph info: %w", err)
	}
	return result, nil
}

// LooseObjectsInfo contains information about loose objects.
type LooseObjectsInfo struct {
	// Count is the number of loose objects.
	Count uint64
	// Size is the total size of all loose objects in bytes.
	Size uint64
	// StaleCount is the number of stale loose objects when taking into account the specified cutoff
	// date.
	StaleCount uint64
	// StaleSize is the total size of stale loose objects when taking into account the specified
	// cutoff date.
	StaleSize uint64
	// GarbageCount is the number of garbage files in the loose-objects shards.
	GarbageCount uint64
	// GarbageSize is the total size of garbage in the loose-objects shards.
	GarbageSize uint64
}

// LoadLooseObjectsInfo collects all loose objects information.
//
//nolint:gosec
func LoadLooseObjectsInfo(repoPath string, cutoffDate time.Time) (LooseObjectsInfo, error) {
	objectsDir := filepath.Join(repoPath, "objects")
	var result LooseObjectsInfo

	subdirs, err := os.ReadDir(objectsDir)
	if err != nil {
		return LooseObjectsInfo{}, fmt.Errorf("reading objects dir: %w", err)
	}

	for _, subdir := range subdirs {
		if !subdir.IsDir() || len(subdir.Name()) != 2 {
			continue // skip invalid loose object dirs
		}

		subdirPath := filepath.Join(objectsDir, subdir.Name())
		entries, err := os.ReadDir(subdirPath)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return LooseObjectsInfo{}, fmt.Errorf("reading %s: %w", subdirPath, err)
		}

		for _, entry := range entries {
			info, err := entry.Info()
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			if err != nil {
				return LooseObjectsInfo{}, fmt.Errorf("failed to read loose object entry %s: %w", entry.Name(), err)
			}

			if !isValidLooseObjectName(entry.Name()) {
				result.GarbageCount++
				result.GarbageSize += uint64(info.Size())
				continue
			}

			if info.ModTime().Before(cutoffDate) {
				result.StaleCount++
				result.StaleSize += uint64(info.Size())
			}

			result.Count++
			result.Size += uint64(info.Size())
		}
	}
	return result, nil
}

func isValidLooseObjectName(s string) bool {
	for _, c := range []byte(s) {
		if strings.IndexByte("0123456789abcdef", c) < 0 {
			return false
		}
	}
	return true
}

// PackFilesInfo contains information about git pack files.
type PackFilesInfo struct {
	// Count is the number of pack files.
	Count uint64
	// Size is the total size of all pack files in bytes.
	Size uint64
	// ReverseIndexCount is the number of reverse indices.
	ReverseIndexCount uint64
	// CruftCount is the number of cruft pack files which have a .mtimes file.
	CruftCount uint64
	// CruftSize is the size of cruft pack files which have a .mtimes file.
	CruftSize uint64
	// KeepCount is the number of .keep pack files.
	KeepCount uint64
	// KeepSize is the size of .keep pack files.
	KeepSize uint64
	// GarbageCount is the number of garbage files.
	GarbageCount uint64
	// GarbageSize is the total size of all garbage files in bytes.
	GarbageSize uint64
	// Bitmap contains information about the bitmap, if any exists.
	Bitmap BitmapInfo
	// MultiPackIndex contains information about the multi-pack-index, if any exists.
	MultiPackIndex MultiPackIndexInfo
	// MultiPackIndexBitmap contains information about the bitmap for the multi-pack-index, if
	// any exists.
	MultiPackIndexBitmap BitmapInfo
	// LastFullRepack indicates the last date at which a full repack has been performed. If the
	// date cannot be determined then this file is set to the zero time.
	LastFullRepack time.Time
}

// LoadPackFilesInfo loads information about git pack files.
func LoadPackFilesInfo(repoPath string) (PackFilesInfo, error) {
	packsDir := path.Join(repoPath, "objects", "pack")

	entries, err := os.ReadDir(packsDir)
	if errors.Is(err, fs.ErrNotExist) {
		return PackFilesInfo{}, nil
	}
	if err != nil {
		return PackFilesInfo{}, fmt.Errorf("failed to read pack directory %s: %w", packsDir, err)
	}

	result := PackFilesInfo{}

	for _, entry := range entries {
		filename := entry.Name()

		info, err := entry.Info()
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return PackFilesInfo{}, fmt.Errorf("failed to read pack entry %s: %w", entry.Name(), err)
		}

		switch {
		case hasPrefixAndSuffix(filename, "pack-", ".pack"):
			result.Count++
			result.Size += uint64(info.Size()) //nolint:gosec
		case hasPrefixAndSuffix(filename, "pack-", ".keep"):
			result.KeepCount++
			result.KeepSize += uint64(info.Size()) //nolint:gosec
		case hasPrefixAndSuffix(filename, "pack-", ".mtimes"):
			result.CruftCount++
			result.CruftSize += uint64(info.Size()) //nolint:gosec
		case hasPrefixAndSuffix(filename, "pack-", ".rev"):
			result.ReverseIndexCount++
		case hasPrefixAndSuffix(filename, "pack-", ".bitmap"):
			result.Bitmap, err = LoadBitmapInfo(path.Join(packsDir, filename))
			if err != nil {
				return PackFilesInfo{}, fmt.Errorf("failed to read pack bitmap %s: %w", entry.Name(), err)
			}
		case hasPrefixAndSuffix(filename, "multi-pack-index-", ".bitmap"):
			result.MultiPackIndexBitmap, err = LoadBitmapInfo(path.Join(packsDir, filename))
			if err != nil {
				return PackFilesInfo{}, fmt.Errorf("failed to read multi-pack-index bitmap %s: %w", entry.Name(), err)
			}
		case filename == "multi-pack-index":
			result.MultiPackIndex, err = LoadMultiPackIndexInfo(path.Join(packsDir, filename))
			if err != nil {
				return PackFilesInfo{}, fmt.Errorf("failed to read multi-pack-index: %w", err)
			}
		default:
			result.GarbageCount++
			result.GarbageSize += uint64(info.Size()) //nolint:gosec
		}
	}

	lastFullRepack, err := GetLastFullRepackTime(repoPath)
	if err != nil {
		return PackFilesInfo{}, fmt.Errorf("failed to get last full repack time: %w", err)
	}

	result.LastFullRepack = lastFullRepack

	return result, nil
}

func SetLastFullRepackTime(repoPath string, t time.Time) error {
	fullpath := filepath.Join(repoPath, FullRepackTimestampFileName)

	err := os.Chtimes(fullpath, t, t)
	if err == nil {
		return nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	f, err := os.CreateTemp(repoPath, FullRepackTimestampFileName+"-*")
	if err != nil {
		return err
	}
	defer func() {
		_ = os.Remove(f.Name())
	}()

	if err := os.Chtimes(f.Name(), t, t); err != nil {
		return err
	}

	if err := os.Rename(f.Name(), fullpath); err != nil {
		return err
	}

	return nil
}

// GetLastFullRepackTime returns the last date at which a full repack has been performed.
func GetLastFullRepackTime(repoPath string) (time.Time, error) {
	info, err := os.Stat(path.Join(repoPath, FullRepackTimestampFileName))
	if errors.Is(err, fs.ErrNotExist) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

func hasPrefixAndSuffix(s, prefix, suffix string) bool {
	return strings.HasPrefix(s, prefix) && strings.HasSuffix(s, suffix)
}

type BitmapInfo struct {
	// Version is the version of the bitmap. Expected always to be 1.
	Version uint16
	// HasHashCache indicates whether the hash cache exists.
	// [GIT]: https://git-scm.com/docs/bitmap-format#_name_hash_cache
	HasHashCache bool
	// HasLookupTable indicates whether the lookup table exists.
	// [GIT]: https://git-scm.com/docs/bitmap-format#_commit_lookup_table
	HasLookupTable bool
}

// LoadBitmapInfo loads information about the git pack bitmap file.
func LoadBitmapInfo(filename string) (BitmapInfo, error) {
	// The bitmap header is defined in
	// https://git-scm.com/docs/bitmap-format#_on_disk_format
	header := []byte{
		0, 0, 0, 0, // 4-byte signature
		0, 0, // 2-byte version number (network byte order), Git only writes or recognizes version 1
		0, 0, // 2-byte flags (network byte order)
	}

	f, err := os.Open(filename)
	if err != nil {
		return BitmapInfo{}, fmt.Errorf("failed to open bitmap file %s: %w", filename, err)
	}
	defer f.Close()

	_, err = io.ReadFull(f, header)
	if err != nil {
		return BitmapInfo{}, fmt.Errorf("failed to read bitmap file %s: %w", filename, err)
	}

	if !bytes.Equal(header[0:4], []byte{'B', 'I', 'T', 'M'}) {
		return BitmapInfo{}, fmt.Errorf("invalid bitmap file signature: %s", filename)
	}

	version := binary.BigEndian.Uint16(header[4:6])
	if version != 1 {
		return BitmapInfo{}, fmt.Errorf("unsupported bitmap file version: %s", filename)
	}

	flags := binary.BigEndian.Uint16(header[6:8])

	return BitmapInfo{
		Version: version,
		// https://git-scm.com/docs/bitmap-format#Documentation/technical/bitmap-format.txt-BITMAPOPTHASHCACHE0x4
		HasHashCache: flags&0x04 == 0x04, // BITMAP_OPT_HASH_CACHE (0x4)
		// https://git-scm.com/docs/bitmap-format#Documentation/technical/bitmap-format.txt-BITMAPOPTLOOKUPTABLE0x10
		HasLookupTable: flags&0x10 == 0x10, // BITMAP_OPT_LOOKUP_TABLE (0x10)
	}, nil
}

type MultiPackIndexInfo struct {
	Exists        bool
	Version       uint8
	PackFileCount uint64
}

// LoadMultiPackIndexInfo loads information about the git multi-pack-index file.
func LoadMultiPackIndexInfo(filename string) (MultiPackIndexInfo, error) {
	// The header is defined in
	// https://git-scm.com/docs/gitformat-pack#_multi_pack_index_midx_files_have_the_following_format
	header := []byte{
		0, 0, 0, 0, // 4-byte signature
		0,          // 1-byte version number, Git only writes or recognizes version 1
		0,          // 1-byte Object ID version
		0,          // 1-byte number of chunks
		0,          // 1-byte number of base multi-pack-index files
		0, 0, 0, 0, // 4-byte number of pack files
	}

	f, err := os.Open(filename)
	if err != nil {
		return MultiPackIndexInfo{}, fmt.Errorf("failed to open multi-pack-index file %s: %w", filename, err)
	}
	defer f.Close()

	_, err = io.ReadFull(f, header)
	if err != nil {
		return MultiPackIndexInfo{}, fmt.Errorf("failed to read multi-pack-index file %s: %w", filename, err)
	}

	if !bytes.Equal(header[0:4], []byte{'M', 'I', 'D', 'X'}) {
		return MultiPackIndexInfo{}, fmt.Errorf("invalid multi-pack-index file signature: %s", filename)
	}

	version := header[4]
	if version != 1 {
		return MultiPackIndexInfo{}, fmt.Errorf("unsupported multi-pack-index file version: %s", filename)
	}

	// The number of base multi-pack-index files is always 0 in Git.
	baseFilesCount := header[7]
	if baseFilesCount != 0 {
		return MultiPackIndexInfo{}, fmt.Errorf("unsupported multi-pack-index file base files count: %d", baseFilesCount)
	}

	packFilesCount := binary.BigEndian.Uint32(header[8:12])

	return MultiPackIndexInfo{
		Exists:        true,
		Version:       version,
		PackFileCount: uint64(packFilesCount),
	}, nil
}

type ReferencesInfo struct {
	LooseReferenceCount uint64
	PackedReferenceSize uint64
}

func LoadReferencesInfo(repoPath string) (ReferencesInfo, error) {
	refsPath := path.Join(repoPath, "refs")
	result := ReferencesInfo{}
	if err := filepath.WalkDir(refsPath, func(_ string, d fs.DirEntry, err error) error {
		if errors.Is(err, fs.ErrNotExist) {
			// ignore if it doesn't exist
			return nil
		}
		if err != nil {
			return err
		}

		if !d.IsDir() {
			result.LooseReferenceCount++
		}

		return nil
	}); err != nil {
		return ReferencesInfo{}, err
	}

	stats, err := os.Stat(path.Join(repoPath, "packed-refs"))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return ReferencesInfo{}, fmt.Errorf("failed to get packed refs size %s: %w", repoPath, err)
	}
	if stats != nil {
		result.PackedReferenceSize = uint64(stats.Size()) //nolint:gosec
	}

	return result, nil
}

// CommitGraphInfo returns information about the commit-graph of a repository.
type CommitGraphInfo struct {
	// Exists tells whether the repository has a commit-graph.
	Exists bool
	// ChainLength is the length of the commit-graph chain, if it exists. If the
	// repository does not have a commit-graph chain but a monolithic commit-graph, then this
	// field will be set to 0.
	ChainLength uint64
	// HasBloomFilters tells whether the commit-graph has bloom filters. Bloom filters are used
	// to answer whether a certain path has been changed in the commit the bloom
	// filter applies to.
	HasBloomFilters bool
	// HasGenerationData tells whether the commit-graph has generation data. Generation
	// data is stored as the corrected committer date, which is defined as the maximum
	// of the commit's own committer date or the corrected committer date of any of its
	// parents. This data can be used to determine whether a commit A comes after a
	// certain commit B.
	HasGenerationData bool
	// HasGenerationDataOverflow stores overflow data in case the corrected committer
	// date takes more than 31 bits to represent.
	HasGenerationDataOverflow bool
}

// LoadCommitGraphInfo returns information about the commit-graph of a repository.
func LoadCommitGraphInfo(repoPath string) (CommitGraphInfo, error) {
	var info CommitGraphInfo

	commitGraphChainPath := filepath.Join(
		repoPath,
		"objects",
		"info",
		"commit-graphs",
		"commit-graph-chain",
	)

	var commitGraphPaths []string

	chainData, err := os.ReadFile(commitGraphChainPath)
	//nolint:nestif
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return CommitGraphInfo{}, fmt.Errorf("reading commit-graphs chain: %w", err)
		}
		// If we couldn't find it, we check whether the monolithic commit-graph file exists
		// and use that instead.
		commitGraphPath := filepath.Join(
			repoPath,
			"objects",
			"info",
			"commit-graph",
		)
		if _, err := os.Stat(commitGraphPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return CommitGraphInfo{Exists: false}, nil
			}

			return CommitGraphInfo{}, fmt.Errorf("statting commit-graph: %w", err)
		}

		commitGraphPaths = []string{commitGraphPath}

		info.Exists = true
	} else {
		// Otherwise, if we have found the commit-graph-chain, we use the IDs it contains as
		// the set of commit-graphs to check further down below.
		ids := bytes.Split(bytes.TrimSpace(chainData), []byte{'\n'})

		commitGraphPaths = make([]string, 0, len(ids))
		for _, id := range ids {
			commitGraphPaths = append(commitGraphPaths,
				filepath.Join(repoPath, "objects", "info", "commit-graphs", fmt.Sprintf("graph-%s.graph", id)),
			)
		}

		info.Exists = true
		info.ChainLength = uint64(len(commitGraphPaths))
	}

	for _, graphFilePath := range commitGraphPaths {
		err := parseCommitGraphFile(graphFilePath, &info)
		if errors.Is(err, os.ErrNotExist) {
			// concurrently modified
			continue
		}
		if err != nil {
			return CommitGraphInfo{}, err
		}
	}

	return info, nil
}

func parseCommitGraphFile(filename string, info *CommitGraphInfo) error {
	f, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("read commit graph chain file: %w", err)
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	// The header format is defined in gitformat-commit-graph(5).
	header := []byte{
		0, 0, 0, 0, // 4-byte signature: The signature is: {'C', 'G', 'P', 'H'}
		0, // 1-byte version number: Currently, the only valid version is 1.
		0, // 1-byte Hash Version
		0, // 1-byte number of "chunks"
		0, // 1-byte number of base commit-graphs
	}

	n, err := reader.Read(header)
	if err != nil {
		return fmt.Errorf("read commit graph file %q header: %w", filename, err)
	}
	if n != len(header) {
		return fmt.Errorf("commit graph file %q is too small, no header", filename)
	}

	if !bytes.Equal(header[:4], []byte("CGPH")) {
		return fmt.Errorf("commit graph file %q doesn't have signature", filename)
	}
	if header[4] != 1 {
		return fmt.Errorf(
			"commit graph file %q has unsupported version number: %v", filename, header[4])
	}

	const chunkTableEntrySize = 12
	numberOfChunks := header[6]
	table := make([]byte, (numberOfChunks+1)*chunkTableEntrySize)

	n, err = reader.Read(table)
	if err != nil {
		return fmt.Errorf(
			"read commit graph file %q table of contents for the chunks: %w", filename, err)
	}

	if n != len(table) {
		return fmt.Errorf(
			"commit graph file %q is too small, no table of contents", filename)
	}

	if !info.HasBloomFilters {
		info.HasBloomFilters = bytes.Contains(table, []byte("BIDX")) && bytes.Contains(table, []byte("BDAT"))
	}

	if !info.HasGenerationData {
		info.HasGenerationData = bytes.Contains(table, []byte("GDA2"))
	}

	if !info.HasGenerationDataOverflow {
		info.HasGenerationDataOverflow = bytes.Contains(table, []byte("GDO2"))
	}
	return nil
}
