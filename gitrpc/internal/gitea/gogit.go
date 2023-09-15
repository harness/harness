// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitea

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/gitrpc/internal/types"

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

func NewGoGitRepoProvider(objectCacheMax int, cacheDuration time.Duration) *GoGitRepoProvider {
	c := cache.New[string, *gogitcache.ObjectLRU](gitObjectCacheGetter{
		maxSize: objectCacheMax,
	}, cacheDuration)
	return &GoGitRepoProvider{
		gitObjectCache: c,
	}
}

func (gr *GoGitRepoProvider) Get(ctx context.Context, path string) (*gogit.Repository, error) {
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

func (r gitObjectCacheGetter) Find(_ context.Context, _ string) (*gogitcache.ObjectLRU, error) {
	return gogitcache.NewObjectLRU(gogitcache.FileSize(r.maxSize)), nil
}

func (g Adapter) getGoGitCommit(ctx context.Context,
	repoPath string,
	rev string,
) (*gogit.Repository, *gogitobject.Commit, error) {
	repo, err := g.repoProvider.Get(ctx, repoPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open repository: %w", err)
	}

	var refSHA *gogitplumbing.Hash
	if rev == "" {
		var head *gogitplumbing.Reference
		head, err = repo.Head()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get head: %w", err)
		}

		headHash := head.Hash()
		refSHA = &headHash
	} else {
		refSHA, err = repo.ResolveRevision(gogitplumbing.Revision(rev))
		if errors.Is(err, gogitplumbing.ErrReferenceNotFound) {
			return nil, nil, types.ErrNotFound
		} else if err != nil {
			return nil, nil, fmt.Errorf("failed to resolve revision %s: %w", rev, err)
		}
	}

	refCommit, err := repo.CommitObject(*refSHA)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load commit data: %w", err)
	}

	return repo, refCommit, nil
}
