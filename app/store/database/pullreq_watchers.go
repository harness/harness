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

package database

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var _ store.PullReqWatcherStore = (*PullReqWatcherStore)(nil)

// NewPullReqWatcherStore returns a new PullReqWatcherStore.
func NewPullReqWatcherStore(db *sqlx.DB, pCache store.PrincipalInfoCache) *PullReqWatcherStore {
	return &PullReqWatcherStore{
		db:     db,
		pCache: pCache,
	}
}

// PullReqWatcherStore implements store.PullReqWatcherStore backed by a relational database.
type PullReqWatcherStore struct {
	db     *sqlx.DB
	pCache store.PrincipalInfoCache
}

// pullReqWatcher is used to fetch pull request watcher data from the database.
type pullReqWatcher struct {
	PullReqID   int64 `db:"pullreq_watcher_pullreq_id"`
	PrincipalID int64 `db:"pullreq_watcher_principal_id"`
	Created     int64 `db:"pullreq_watcher_created"`
}

const pullreqWatcherColumns = `
	 pullreq_watcher_pullreq_id
	,pullreq_watcher_principal_id
	,pullreq_watcher_created`

// Create adds the principal to the watchers of the pull request. It's a no-op if already watching.
func (s *PullReqWatcherStore) Create(ctx context.Context, prID, principalID int64) error {
	const sqlQuery = `
	INSERT INTO pullreq_watchers (
		 pullreq_watcher_pullreq_id
		,pullreq_watcher_principal_id
		,pullreq_watcher_created
	) VALUES ($1, $2, $3)
	ON CONFLICT (pullreq_watcher_pullreq_id, pullreq_watcher_principal_id) DO NOTHING`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sqlQuery, prID, principalID, time.Now().UnixMilli()); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to insert pull request watcher")
	}

	return nil
}

// Delete removes the principal from the watchers of the pull request. It's a no-op if not watching.
func (s *PullReqWatcherStore) Delete(ctx context.Context, prID, principalID int64) error {
	const sqlQuery = `
	DELETE FROM pullreq_watchers
	WHERE pullreq_watcher_pullreq_id = $1 AND
	      pullreq_watcher_principal_id = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sqlQuery, prID, principalID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to delete pull request watcher")
	}

	return nil
}

// Exists returns true if the principal is watching the pull request.
func (s *PullReqWatcherStore) Exists(ctx context.Context, prID, principalID int64) (bool, error) {
	const sqlQuery = `
	SELECT COUNT(*)
	FROM pullreq_watchers
	WHERE pullreq_watcher_pullreq_id = $1 AND
	      pullreq_watcher_principal_id = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	if err := db.GetContext(ctx, &count, sqlQuery, prID, principalID); err != nil {
		return false, database.ProcessSQLErrorf(ctx, err, "Failed to check pull request watcher existence")
	}

	return count > 0, nil
}

// List returns the watchers of the pull request, oldest first.
func (s *PullReqWatcherStore) List(ctx context.Context, prID int64) ([]*types.PullReqWatcher, error) {
	stmt := database.Builder.
		Select(pullreqWatcherColumns).
		From("pullreq_watchers").
		Where("pullreq_watcher_pullreq_id = ?", prID).
		OrderBy("pullreq_watcher_created asc").
		// Memory safety net - a pull request shouldn't have more watchers than this.
		Limit(types.MaxPullReqWatchers)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert pull request watcher list query to sql")
	}

	dst := make([]*pullReqWatcher, 0)

	db := dbtx.GetAccessor(ctx, s.db)

	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed executing pull request watcher list query")
	}

	return s.mapSlicePullReqWatcher(ctx, dst)
}

// Count returns the number of watchers of the pull request.
func (s *PullReqWatcherStore) Count(ctx context.Context, prID int64) (int64, error) {
	stmt := database.Builder.
		Select("COUNT(*)").
		From("pullreq_watchers").
		Where("pullreq_watcher_pullreq_id = ?", prID)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return 0, errors.Wrap(err, "Failed to convert pull request watcher count query to sql")
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var count int64
	if err = db.GetContext(ctx, &count, sql, args...); err != nil {
		return 0, database.ProcessSQLErrorf(ctx, err, "Failed executing pull request watcher count query")
	}

	return count, nil
}

func (s *PullReqWatcherStore) mapSlicePullReqWatcher(
	ctx context.Context,
	watchers []*pullReqWatcher,
) ([]*types.PullReqWatcher, error) {
	ids := make([]int64, len(watchers))
	for i, v := range watchers {
		ids[i] = v.PrincipalID
	}

	infoMap, err := s.pCache.Map(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to load PR watcher principal infos: %w", err)
	}

	m := make([]*types.PullReqWatcher, len(watchers))
	for i, v := range watchers {
		m[i] = &types.PullReqWatcher{
			PullReqID:   v.PullReqID,
			PrincipalID: v.PrincipalID,
			Created:     v.Created,
		}
		if info, ok := infoMap[v.PrincipalID]; ok {
			m[i].Watcher = *info
		}
	}

	return m, nil
}
