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

package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/app/services/locker"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/lock"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func crpNewLocker() *locker.Locker {
	return locker.NewLocker(lock.NewInMemory(lock.Config{
		App:        "test",
		Namespace:  "test",
		Expiry:     time.Minute,
		Tries:      3,
		RetryDelay: 10 * time.Millisecond,
	}))
}

// crpRepoStore is an in-memory store.RepoStore recording the calls it receives.
// Errors can be injected per method name through errs.
type crpRepoStore struct {
	store.RepoStore
	repos  map[int64]*types.Repository
	errs   map[string]error
	calls  []string
	nextID int64
	// restored holds the repository handed back by Restore.
	restored *types.Repository
}

func newCrpRepoStore(repos ...*types.Repository) *crpRepoStore {
	s := &crpRepoStore{
		repos:  map[int64]*types.Repository{},
		errs:   map[string]error{},
		nextID: 100,
	}
	for _, r := range repos {
		s.repos[r.ID] = r
	}

	return s
}

func (s *crpRepoStore) record(name string) error {
	s.calls = append(s.calls, name)

	return s.errs[name]
}

func (s *crpRepoStore) called(name string) bool {
	for _, c := range s.calls {
		if c == name {
			return true
		}
	}

	return false
}

func (s *crpRepoStore) Find(_ context.Context, id int64) (*types.Repository, error) {
	if err := s.record("Find"); err != nil {
		return nil, err
	}

	repo, ok := s.repos[id]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}

	clone := repo.Clone()

	return &clone, nil
}

func (s *crpRepoStore) FindDeleted(_ context.Context, id int64, _ *int64) (*types.Repository, error) {
	if err := s.record("FindDeleted"); err != nil {
		return nil, err
	}

	repo, ok := s.repos[id]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}

	clone := repo.Clone()

	return &clone, nil
}

func (s *crpRepoStore) FindActiveByUID(_ context.Context, parentID int64, uid string) (*types.Repository, error) {
	if err := s.record("FindActiveByUID"); err != nil {
		return nil, err
	}

	for _, repo := range s.repos {
		if repo.ParentID == parentID && repo.Identifier == uid && repo.Deleted == nil {
			clone := repo.Clone()

			return &clone, nil
		}
	}

	return nil, gitnessstore.ErrResourceNotFound
}

func (s *crpRepoStore) Create(_ context.Context, repo *types.Repository) error {
	if err := s.record("Create"); err != nil {
		return err
	}

	s.nextID++
	repo.ID = s.nextID
	repo.Path = fmt.Sprintf("space/%s", repo.Identifier)
	s.repos[repo.ID] = repo

	return nil
}

func (s *crpRepoStore) UpdateOptLock(
	_ context.Context,
	repo *types.Repository,
	mutate func(*types.Repository) error,
) (*types.Repository, error) {
	if err := s.record("UpdateOptLock"); err != nil {
		return nil, err
	}

	clone := repo.Clone()
	if err := mutate(&clone); err != nil {
		return nil, err
	}

	s.repos[clone.ID] = &clone

	return &clone, nil
}

func (s *crpRepoStore) SoftDelete(_ context.Context, repo *types.Repository, deletedAt int64) error {
	if err := s.record("SoftDelete"); err != nil {
		return err
	}

	repo.Deleted = &deletedAt

	return nil
}

func (s *crpRepoStore) Purge(_ context.Context, id int64, _ *int64) error {
	if err := s.record("Purge"); err != nil {
		return err
	}

	delete(s.repos, id)

	return nil
}

func (s *crpRepoStore) ClearForkID(context.Context, int64) error { return s.record("ClearForkID") }

func (s *crpRepoStore) UpdateNumForks(context.Context, int64, int64) error {
	return s.record("UpdateNumForks")
}

func (s *crpRepoStore) UpdateRootSpace(context.Context, []int64, int64, string) (err error) {
	return s.record("UpdateRootSpace")
}

func (s *crpRepoStore) Restore(
	_ context.Context, repo *types.Repository, newIdentifier *string, newParentID *int64,
) (*types.Repository, error) {
	if err := s.record("Restore"); err != nil {
		return nil, err
	}

	restored := repo.Clone()
	restored.Deleted = nil
	if newIdentifier != nil {
		restored.Identifier = *newIdentifier
	}
	if newParentID != nil {
		restored.ParentID = *newParentID
	}
	s.restored = &restored

	return &restored, nil
}

// crpSpaceIDCache is an in-memory store.SpaceIDCache.
type crpSpaceIDCache struct {
	spaces map[int64]*types.SpaceCore
}

func (c *crpSpaceIDCache) Stats() (int64, int64) { return 0, 0 }

func (c *crpSpaceIDCache) Get(_ context.Context, id int64) (*types.SpaceCore, error) {
	space, ok := c.spaces[id]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}

	clone := *space

	return &clone, nil
}

func (c *crpSpaceIDCache) Evict(context.Context, int64) {}

var _ store.SpaceIDCache = (*crpSpaceIDCache)(nil)

// crpSpaceStore is a store.SpaceStore serving spaces from memory.
type crpSpaceStore struct {
	store.SpaceStore
	spaces      map[int64]*types.Space
	ancestors   []int64
	ancestorErr error
	err         error
}

func (s *crpSpaceStore) get(id int64) (*types.Space, error) {
	if s.err != nil {
		return nil, s.err
	}

	space, ok := s.spaces[id]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}

	return space, nil
}

func (s *crpSpaceStore) Find(_ context.Context, id int64) (*types.Space, error) { return s.get(id) }

func (s *crpSpaceStore) FindForUpdate(_ context.Context, id int64) (*types.Space, error) {
	return s.get(id)
}

func (s *crpSpaceStore) FindByRef(_ context.Context, ref string) (*types.Space, error) {
	if s.err != nil {
		return nil, s.err
	}

	for _, space := range s.spaces {
		if space.Path == ref {
			return space, nil
		}
	}

	return nil, gitnessstore.ErrResourceNotFound
}

func (s *crpSpaceStore) GetAncestorIDs(context.Context, int64) ([]int64, error) {
	return s.ancestors, s.ancestorErr
}

// crpPublicAccess is a publicaccess.Service backed by a map with injectable errors.
type crpPublicAccess struct {
	publicaccess.Service
	public       map[string]bool
	unsupported  bool
	getErr       error
	setErr       error
	deleteErr    error
	supportedErr error
	deleted      []string
}

func newCrpPublicAccess() *crpPublicAccess {
	return &crpPublicAccess{public: map[string]bool{}}
}

func (p *crpPublicAccess) Get(_ context.Context, _ enum.PublicResourceType, path string) (bool, error) {
	return p.public[path], p.getErr
}

func (p *crpPublicAccess) Set(_ context.Context, _ enum.PublicResourceType, path string, enable bool) error {
	if p.setErr != nil {
		return p.setErr
	}

	p.public[path] = enable

	return nil
}

func (p *crpPublicAccess) Delete(_ context.Context, _ enum.PublicResourceType, path string) error {
	if p.deleteErr != nil {
		return p.deleteErr
	}

	p.deleted = append(p.deleted, path)
	delete(p.public, path)

	return nil
}

func (p *crpPublicAccess) IsPublicAccessSupported(
	context.Context, enum.PublicResourceType, string,
) (bool, error) {
	return !p.unsupported, p.supportedErr
}

// crpFavoriteStore is a store.FavoriteStore with a fixed favorites map.
type crpFavoriteStore struct {
	store.FavoriteStore
	favorites map[int64]bool
	err       error
}

func (s crpFavoriteStore) Map(context.Context, int64, enum.ResourceType, []int64) (map[int64]bool, error) {
	return s.favorites, s.err
}

// crpIndexer is a keywordsearch.Indexer recording the indexed repositories.
type crpIndexer struct {
	indexed []int64
	err     error
}

func (i *crpIndexer) Index(_ context.Context, repo *types.Repository) error {
	i.indexed = append(i.indexed, repo.ID)

	return i.err
}
