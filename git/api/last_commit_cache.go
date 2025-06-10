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
	"context"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/sha"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
)

func NewInMemoryLastCommitCache(
	cacheDuration time.Duration,
) cache.Cache[CommitEntryKey, *Commit] {
	return cache.New[CommitEntryKey, *Commit](
		commitEntryGetter{},
		cacheDuration)
}

func logCacheErrFn(ctx context.Context, err error) {
	log.Ctx(ctx).Warn().Msgf("failed to use cache: %s", err.Error())
}

func NewRedisLastCommitCache(
	redisClient redis.UniversalClient,
	cacheDuration time.Duration,
) (cache.Cache[CommitEntryKey, *Commit], error) {
	if redisClient == nil {
		return nil, errors.New("unable to create redis based LastCommitCache as redis client is nil")
	}

	return cache.NewRedis[CommitEntryKey, *Commit](
		redisClient,
		commitEntryGetter{},
		func(key CommitEntryKey) string {
			h := sha256.New()
			h.Write([]byte(key))
			return "last_commit:" + hex.EncodeToString(h.Sum(nil))
		},
		commitValueCodec{},
		cacheDuration,
		logCacheErrFn,
	), nil
}

func NoLastCommitCache() cache.Cache[CommitEntryKey, *Commit] {
	return cache.NewNoCache[CommitEntryKey, *Commit](commitEntryGetter{})
}

type CommitEntryKey string

const separatorZero = "\x00"

func makeCommitEntryKey(
	repoPath string,
	commitSHA sha.SHA,
	path string,
) CommitEntryKey {
	return CommitEntryKey(repoPath + separatorZero + commitSHA.String() + separatorZero + path)
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

func (c commitValueCodec) Encode(v *Commit) string {
	buffer := &strings.Builder{}
	_ = gob.NewEncoder(buffer).Encode(v)
	return buffer.String()
}

func (c commitValueCodec) Decode(s string) (*Commit, error) {
	commit := &Commit{}
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
) (*Commit, error) {
	repoPath, commitSHA, path := key.Split()

	if path == "" {
		path = "."
	}

	return GetLatestCommit(ctx, repoPath, commitSHA, path)
}
