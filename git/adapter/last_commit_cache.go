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
	"strings"
	"time"

	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/git/types"

	gitea "code.gitea.io/gitea/modules/git"
	gogitplumbing "github.com/go-git/go-git/v5/plumbing"
	"github.com/go-redis/redis/v8"
)

func NewInMemoryLastCommitCache(
	cacheDuration time.Duration,
	repoProvider *GoGitRepoProvider,
) cache.Cache[CommitEntryKey, *types.Commit] {
	return cache.New[CommitEntryKey, *types.Commit](
		commitEntryGetter{
			repoProvider: repoProvider,
		},
		cacheDuration)
}

func NewRedisLastCommitCache(
	redisClient redis.UniversalClient,
	cacheDuration time.Duration,
	repoProvider *GoGitRepoProvider,
) cache.Cache[CommitEntryKey, *types.Commit] {
	return cache.NewRedis[CommitEntryKey, *types.Commit](
		redisClient,
		commitEntryGetter{
			repoProvider: repoProvider,
		},
		func(key CommitEntryKey) string {
			h := sha256.New()
			h.Write([]byte(key))
			return "last_commit:" + hex.EncodeToString(h.Sum(nil))
		},
		commitValueCodec{},
		cacheDuration)
}

func NoLastCommitCache(
	repoProvider *GoGitRepoProvider,
) cache.Cache[CommitEntryKey, *types.Commit] {
	return cache.NewNoCache[CommitEntryKey, *types.Commit](commitEntryGetter{repoProvider: repoProvider})
}

type CommitEntryKey string

const commitEntryKeySeparator = "\x00"

func makeCommitEntryKey(
	repoPath string,
	commitSHA string,
	path string,
) CommitEntryKey {
	return CommitEntryKey(repoPath + commitEntryKeySeparator + commitSHA + commitEntryKeySeparator + path)
}

func (c CommitEntryKey) Split() (
	repoPath string,
	commitSHA string,
	path string,
) {
	parts := strings.Split(string(c), commitEntryKeySeparator)
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

type commitEntryGetter struct {
	repoProvider *GoGitRepoProvider
}

// Find implements the cache.Getter interface.
func (c commitEntryGetter) Find(
	ctx context.Context,
	key CommitEntryKey,
) (*types.Commit, error) {
	repoPath, rev, path := key.Split()

	if path == "" {
		path = "."
	}

	args := []string{"log", "--max-count=1", "--format=%H", rev, "--", path}
	commitSHA, _, runErr := gitea.NewCommand(ctx, args...).RunStdString(&gitea.RunOpts{Dir: repoPath})
	if runErr != nil {
		return nil, fmt.Errorf("failed to run git: %w", runErr)
	}

	commitSHA = strings.TrimSpace(commitSHA)

	if commitSHA == "" {
		return nil, types.ErrNotFound("revision '%s' not found", rev)
	}

	repo, err := c.repoProvider.Get(ctx, repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository %s from cache: %w", repoPath, err)
	}

	commit, err := repo.CommitObject(gogitplumbing.NewHash(commitSHA))
	if err != nil {
		return nil, fmt.Errorf("failed to load commit data: %w", err)
	}

	var title string
	var message string

	title = commit.Message
	if idx := strings.IndexRune(commit.Message, '\n'); idx >= 0 {
		title = commit.Message[:idx]
		message = commit.Message[idx+1:]
	}

	return &types.Commit{
		SHA:     commitSHA,
		Title:   title,
		Message: message,
		Author: types.Signature{
			Identity: types.Identity{
				Name:  commit.Author.Name,
				Email: commit.Author.Email,
			},
			When: commit.Author.When,
		},
		Committer: types.Signature{
			Identity: types.Identity{
				Name:  commit.Committer.Name,
				Email: commit.Committer.Email,
			},
			When: commit.Committer.When,
		},
	}, nil
}
