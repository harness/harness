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

package adapter

import (
	"context"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/types"

	gitea "code.gitea.io/gitea/modules/git"
	"github.com/go-redis/redis/v8"
)

func NewInMemoryLastCommitCache(
	cacheDuration time.Duration,
) cache.Cache[CommitEntryKey, *types.Commit] {
	return cache.New[CommitEntryKey, *types.Commit](
		commitEntryGetter{},
		cacheDuration)
}

func NewRedisLastCommitCache(
	redisClient redis.UniversalClient,
	cacheDuration time.Duration,
) cache.Cache[CommitEntryKey, *types.Commit] {
	return cache.NewRedis[CommitEntryKey, *types.Commit](
		redisClient,
		commitEntryGetter{},
		func(key CommitEntryKey) string {
			h := sha256.New()
			h.Write([]byte(key))
			return "last_commit:" + hex.EncodeToString(h.Sum(nil))
		},
		commitValueCodec{},
		cacheDuration)
}

func NoLastCommitCache() cache.Cache[CommitEntryKey, *types.Commit] {
	return cache.NewNoCache[CommitEntryKey, *types.Commit](commitEntryGetter{})
}

type CommitEntryKey string

const separatorZero = "\x00"

func makeCommitEntryKey(
	repoPath string,
	commitSHA string,
	path string,
) CommitEntryKey {
	return CommitEntryKey(repoPath + separatorZero + commitSHA + separatorZero + path)
}

func (c CommitEntryKey) Split() (
	repoPath string,
	commitSHA string,
	path string,
) {
	parts := strings.Split(string(c), separatorZero)
	if len(parts) != 3 {
		return
	}

	repoPath = parts[0]
	commitSHA = parts[1]
	path = parts[2]

	return
}

type commitValueCodec struct{}

func (c commitValueCodec) Encode(v *types.Commit) string {
	buffer := &strings.Builder{}
	_ = gob.NewEncoder(buffer).Encode(v)
	return buffer.String()
}

func (c commitValueCodec) Decode(s string) (*types.Commit, error) {
	commit := &types.Commit{}
	if err := gob.NewDecoder(strings.NewReader(s)).Decode(commit); err != nil {
		return nil, fmt.Errorf("failed to unpack commit entry value: %w", err)
	}

	return commit, nil
}

type commitEntryGetter struct{}

// Find implements the cache.Getter interface.
func (c commitEntryGetter) Find(
	ctx context.Context,
	key CommitEntryKey,
) (*types.Commit, error) {
	repoPath, commitSHA, path := key.Split()

	if path == "" {
		path = "."
	}

	const format = "" +
		fmtCommitHash + fmtZero + // 0
		fmtAuthorName + fmtZero + // 1
		fmtAuthorEmail + fmtZero + // 2
		fmtAuthorUnix + fmtZero + // 3
		fmtCommitterName + fmtZero + // 4
		fmtCommitterEmail + fmtZero + // 5
		fmtCommitterUnix + fmtZero + // 6
		fmtSubject + fmtZero + // 7
		fmtBody // 8

	args := []string{"log", "--max-count=1", "--format=" + format, commitSHA, "--", path}
	commitLine, _, err := gitea.NewCommand(ctx, args...).RunStdString(&gitea.RunOpts{Dir: repoPath})
	if err != nil {
		return nil, fmt.Errorf("failed to run git to get the last commit for path: %w", err)
	}

	const columnCount = 9

	commitData := strings.Split(strings.TrimSpace(commitLine), separatorZero)
	if len(commitData) != columnCount {
		return nil, errors.InvalidArgument("path %q not found in commit %s", path, commitSHA)
	}

	sha := commitData[0]
	authorName := commitData[1]
	authorEmail := commitData[2]
	authorTime, _ := strconv.ParseInt(commitData[3], 10, 64) // parse failure produces 01-01-1970
	committerName := commitData[4]
	committerEmail := commitData[5]
	committerTime, _ := strconv.ParseInt(commitData[6], 10, 64) // parse failure produces 01-01-1970
	subject := commitData[7]
	body := commitData[8]

	return &types.Commit{
		SHA:     sha,
		Title:   subject,
		Message: body,
		Author: types.Signature{
			Identity: types.Identity{
				Name:  authorName,
				Email: authorEmail,
			},
			When: time.Unix(authorTime, 0),
		},
		Committer: types.Signature{
			Identity: types.Identity{
				Name:  committerName,
				Email: committerEmail,
			},
			When: time.Unix(committerTime, 0),
		},
	}, nil
}
