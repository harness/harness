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
	"fmt"
	"os"
	"time"

	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/types"

	gogitosfs "github.com/go-git/go-billy/v5/osfs"
	gogit "github.com/go-git/go-git/v5"
	gogitplumbing "github.com/go-git/go-git/v5/plumbing"
	gogitcache "github.com/go-git/go-git/v5/plumbing/cache"
	gogitobject "github.com/go-git/go-git/v5/plumbing/object"
	gogitfilesystem "github.com/go-git/go-git/v5/storage/filesystem"
)

type GoGitRepoProvider struct {
	gitObjectCache cache.Cache[string, *gogitcache.ObjectLRU]
}

func NewGoGitRepoProvider(
	objectCacheMax int,
	cacheDuration time.Duration,
) *GoGitRepoProvider {
	c := cache.New[string, *gogitcache.ObjectLRU](gitObjectCacheGetter{
		maxSize: objectCacheMax,
	}, cacheDuration)
	return &GoGitRepoProvider{
		gitObjectCache: c,
	}
}

func (gr *GoGitRepoProvider) Get(
	ctx context.Context,
	path string,
) (*gogit.Repository, error) {
	fs := gogitosfs.New(path)
	stat, err := fs.Stat("")
	if err != nil {
		if os.IsNotExist(err) {
			return nil, types.ErrRepositoryNotFound
		}

		return nil, fmt.Errorf("failed to check repository existence: %w", err)
	}
	if !stat.IsDir() {
		return nil, types.ErrRepositoryCorrupted
	}

	gitObjectCache, err := gr.gitObjectCache.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository cache: %w", err)
	}

	s := gogitfilesystem.NewStorage(fs, gitObjectCache)

	repo, err := gogit.Open(s, nil)
	if err != nil {
		return nil, err
	}

	return repo, nil
}

type gitObjectCacheGetter struct {
	maxSize int
}

func (r gitObjectCacheGetter) Find(
	_ context.Context,
	_ string,
) (*gogitcache.ObjectLRU, error) {
	return gogitcache.NewObjectLRU(gogitcache.FileSize(r.maxSize)), nil
}

func (a Adapter) getGoGitCommit(
	ctx context.Context,
	repoPath string,
	rev string,
) (*gogit.Repository, *gogitobject.Commit, error) {
	if repoPath == "" {
		return nil, nil, ErrRepositoryPathEmpty
	}
	repo, err := a.repoProvider.Get(ctx, repoPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open repository: %w", err)
	}

	var refSHA *gogitplumbing.Hash
	if rev == "" {
		var head *gogitplumbing.Reference
		head, err = repo.Head()
		if err != nil {
			return nil, nil, errors.Internal("failed to get head: %w", err)
		}

		headHash := head.Hash()
		refSHA = &headHash
	} else {
		refSHA, err = repo.ResolveRevision(gogitplumbing.Revision(rev))
		if errors.Is(err, gogitplumbing.ErrReferenceNotFound) {
			return nil, nil, errors.NotFound("reference not found '%s'", rev)
		} else if err != nil {
			return nil, nil, errors.Internal("failed to resolve revision '%s'", rev, err)
		}
	}

	refCommit, err := repo.CommitObject(*refSHA)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load commit data: %w", err)
	}

	return repo, refCommit, nil
}
