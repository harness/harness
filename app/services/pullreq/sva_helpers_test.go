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

package pullreq

import (
	"context"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	storecache "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// svaRepoIDCache is a store.RepoIDCache backed by a function.
type svaRepoIDCache struct {
	get func(ctx context.Context, id int64) (*types.RepositoryCore, error)
}

func (c *svaRepoIDCache) Stats() (int64, int64) { return 0, 0 }

func (c *svaRepoIDCache) Get(ctx context.Context, id int64) (*types.RepositoryCore, error) {
	return c.get(ctx, id)
}

func (c *svaRepoIDCache) Evict(context.Context, int64) {}

// svaNewRepoFinder builds a repo finder that resolves repositories by ID using the provided function.
func svaNewRepoFinder(
	get func(ctx context.Context, id int64) (*types.RepositoryCore, error),
) refcache.RepoFinder {
	return refcache.NewRepoFinder(
		nil, nil, &svaRepoIDCache{get: get}, nil,
		storecache.Evictor[*types.RepositoryCore]{},
	)
}

// svaFixedRepoFinder builds a repo finder that always resolves to the same repository.
func svaFixedRepoFinder(repo *types.RepositoryCore) refcache.RepoFinder {
	return svaNewRepoFinder(func(context.Context, int64) (*types.RepositoryCore, error) {
		return repo, nil
	})
}

// svaAuthorizer is an authz.Authorizer whose answer is decided by a function.
type svaAuthorizer struct {
	authz.Authorizer

	check func(resource *types.Resource, scope *types.Scope) (bool, error)
}

func (a *svaAuthorizer) Check(
	_ context.Context,
	_ *auth.Session,
	scope *types.Scope,
	resource *types.Resource,
	_ enum.Permission,
) (bool, error) {
	return a.check(resource, scope)
}

// svaPullReqStore overrides the PullReqStore methods used by the tests. Unexpected calls panic.
type svaPullReqStore struct {
	store.PullReqStore

	findByNumber func(ctx context.Context, repoID, number int64) (*types.PullReq, error)
	list         func(ctx context.Context, opts *types.PullReqFilter) ([]*types.PullReq, error)
	count        func(ctx context.Context, opts *types.PullReqFilter) (int64, error)
	stream       func(ctx context.Context, opts *types.PullReqFilter) (<-chan *types.PullReq, <-chan error)
	updateOptLck func(
		ctx context.Context, pr *types.PullReq, fn func(pr *types.PullReq) error,
	) (*types.PullReq, error)
}

func (s *svaPullReqStore) FindByNumber(ctx context.Context, repoID, number int64) (*types.PullReq, error) {
	return s.findByNumber(ctx, repoID, number)
}

func (s *svaPullReqStore) List(ctx context.Context, opts *types.PullReqFilter) ([]*types.PullReq, error) {
	return s.list(ctx, opts)
}

func (s *svaPullReqStore) Count(ctx context.Context, opts *types.PullReqFilter) (int64, error) {
	return s.count(ctx, opts)
}

func (s *svaPullReqStore) Stream(
	ctx context.Context,
	opts *types.PullReqFilter,
) (<-chan *types.PullReq, <-chan error) {
	return s.stream(ctx, opts)
}

func (s *svaPullReqStore) UpdateOptLock(
	ctx context.Context,
	pr *types.PullReq,
	fn func(pr *types.PullReq) error,
) (*types.PullReq, error) {
	return s.updateOptLck(ctx, pr, fn)
}

// svaApplyUpdate is an UpdateOptLock implementation that applies the mutation to the pull request in place.
func svaApplyUpdate(
	_ context.Context,
	pr *types.PullReq,
	fn func(pr *types.PullReq) error,
) (*types.PullReq, error) {
	if err := fn(pr); err != nil {
		return nil, err
	}
	return pr, nil
}

// svaStreamOf returns a stream function emitting the provided pull requests and then the provided error.
func svaStreamOf(
	prs []*types.PullReq,
	streamErr error,
) func(context.Context, *types.PullReqFilter) (<-chan *types.PullReq, <-chan error) {
	return func(context.Context, *types.PullReqFilter) (<-chan *types.PullReq, <-chan error) {
		ch := make(chan *types.PullReq, len(prs))
		for _, pr := range prs {
			ch <- pr
		}
		close(ch)

		chErr := make(chan error, 1)
		chErr <- streamErr
		close(chErr)

		return ch, chErr
	}
}

// svaActivityStore records the payloads written through CreateWithPayload.
type svaActivityStore struct {
	store.PullReqActivityStore

	created []types.PullReqActivityPayload
	seqs    []int64
	err     error
}

func (s *svaActivityStore) CreateWithPayload(
	_ context.Context,
	pr *types.PullReq,
	_ int64,
	payload types.PullReqActivityPayload,
	_ *types.PullReqActivityMetadata,
) (*types.PullReqActivity, error) {
	s.created = append(s.created, payload)
	s.seqs = append(s.seqs, pr.ActivitySeq)
	return nil, s.err
}

// svaRepoStore overrides the RepoStore methods used by the tests.
type svaRepoStore struct {
	store.RepoStore

	find      func(ctx context.Context, id int64) (*types.Repository, error)
	updateOpt func(ctx context.Context, repo *types.Repository, fn func(*types.Repository) error,
	) (*types.Repository, error)
}

func (s *svaRepoStore) Find(ctx context.Context, id int64) (*types.Repository, error) {
	return s.find(ctx, id)
}

func (s *svaRepoStore) UpdateOptLock(
	ctx context.Context,
	repo *types.Repository,
	fn func(*types.Repository) error,
) (*types.Repository, error) {
	return s.updateOpt(ctx, repo, fn)
}

// svaSSE records the published server-sent events.
type svaSSE struct {
	sse.Streamer

	published []any
	spaceIDs  []int64
}

func (s *svaSSE) Publish(_ context.Context, spaceID int64, _ enum.SSEType, data any) {
	s.spaceIDs = append(s.spaceIDs, spaceID)
	s.published = append(s.published, data)
}

// svaSHA builds a deterministic valid SHA made of the repeated hex digit.
func svaSHA(digit string) sha.SHA {
	str := ""
	for range 40 {
		str += digit
	}
	return sha.Must(str)
}
