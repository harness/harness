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
	"time"

	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var _ store.MergeQueueStore = (*MergeQueueStore)(nil)

func NewMergeQueueStore(db *sqlx.DB) *MergeQueueStore {
	return &MergeQueueStore{db: db}
}

type MergeQueueStore struct {
	db *sqlx.DB
}

type mergeQueue struct {
	ID            int64  `db:"merge_queue_id"`
	RepoID        int64  `db:"merge_queue_repo_id"`
	Branch        string `db:"merge_queue_branch"`
	Version       int64  `db:"merge_queue_version"`
	Created       int64  `db:"merge_queue_created"`
	Updated       int64  `db:"merge_queue_updated"`
	OrderSequence int64  `db:"merge_queue_order_sequence"`
}

const (
	mergeQueueColumns = `
		 merge_queue_id
		,merge_queue_repo_id
		,merge_queue_branch
		,merge_queue_version
		,merge_queue_created
		,merge_queue_updated
		,merge_queue_order_sequence`

	mergeQueueSelectBase = `
	SELECT` + mergeQueueColumns + `
	FROM merge_queues`
)

func (s *MergeQueueStore) Find(ctx context.Context, id int64) (*types.MergeQueue, error) {
	const sqlQuery = mergeQueueSelectBase + `
	WHERE merge_queue_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &mergeQueue{}
	if err := db.GetContext(ctx, dst, sqlQuery, id); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find merge queue")
	}

	return s.mapExternal(dst), nil
}

func (s *MergeQueueStore) FindByRepoAndBranch(
	ctx context.Context,
	repoID int64,
	branch string,
) (*types.MergeQueue, error) {
	const sqlQuery = mergeQueueSelectBase + `
	WHERE merge_queue_repo_id = $1 AND merge_queue_branch = $2`

	db := dbtx.GetAccessor(ctx, s.db)

	dst := &mergeQueue{}
	if err := db.GetContext(ctx, dst, sqlQuery, repoID, branch); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "Failed to find merge queue by repo and branch")
	}

	return s.mapExternal(dst), nil
}

func (s *MergeQueueStore) Create(ctx context.Context, q *types.MergeQueue) error {
	const sqlQuery = `
	INSERT INTO merge_queues (
		 merge_queue_repo_id
		,merge_queue_branch
		,merge_queue_version
		,merge_queue_created
		,merge_queue_updated
		,merge_queue_order_sequence
	) values (
		 :merge_queue_repo_id
		,:merge_queue_branch
		,:merge_queue_version
		,:merge_queue_created
		,:merge_queue_updated
		,:merge_queue_order_sequence
	) RETURNING merge_queue_id`

	db := dbtx.GetAccessor(ctx, s.db)

	dbMergeQueue := s.mapInternal(q)

	query, arg, err := db.BindNamed(sqlQuery, dbMergeQueue)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind merge queue object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&q.ID); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to insert merge queue")
	}

	return nil
}

func (s *MergeQueueStore) Update(ctx context.Context, q *types.MergeQueue) error {
	const sqlQuery = `
	UPDATE merge_queues
	SET
		 merge_queue_version = :merge_queue_version
		,merge_queue_updated = :merge_queue_updated
		,merge_queue_order_sequence = :merge_queue_order_sequence
	WHERE merge_queue_id = :merge_queue_id AND merge_queue_version = :merge_queue_version - 1`

	db := dbtx.GetAccessor(ctx, s.db)

	updatedAt := time.Now()

	dbMergeQueue := s.mapInternal(q)

	dbMergeQueue.Version++
	dbMergeQueue.Updated = updatedAt.UnixMilli()

	query, arg, err := db.BindNamed(sqlQuery, dbMergeQueue)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to bind merge queue object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to update merge queue")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return gitness_store.ErrVersionConflict
	}

	updatedMergeQueue := s.mapExternal(dbMergeQueue)
	*q = *updatedMergeQueue

	return nil
}

func (s *MergeQueueStore) Delete(ctx context.Context, id int64) error {
	const sqlQuery = `
	DELETE FROM merge_queues
	WHERE merge_queue_id = $1`

	db := dbtx.GetAccessor(ctx, s.db)

	if _, err := db.ExecContext(ctx, sqlQuery, id); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Failed to delete merge queue")
	}

	return nil
}

func (s *MergeQueueStore) UpdateOptLock(ctx context.Context,
	q *types.MergeQueue,
	mutateFn func(q *types.MergeQueue) error,
) (*types.MergeQueue, error) {
	for {
		dup := *q

		err := mutateFn(&dup)
		if err != nil {
			return nil, err
		}

		err = s.Update(ctx, &dup)
		if err == nil {
			return &dup, nil
		}
		if !errors.Is(err, gitness_store.ErrVersionConflict) {
			return nil, err
		}

		q, err = s.Find(ctx, q.ID)
		if err != nil {
			return nil, err
		}
	}
}

func (*MergeQueueStore) mapExternal(q *mergeQueue) *types.MergeQueue {
	return &types.MergeQueue{
		ID:            q.ID,
		RepoID:        q.RepoID,
		Branch:        q.Branch,
		Version:       q.Version,
		Created:       q.Created,
		Updated:       q.Updated,
		OrderSequence: q.OrderSequence,
	}
}

func (*MergeQueueStore) mapInternal(q *types.MergeQueue) *mergeQueue {
	return &mergeQueue{
		ID:            q.ID,
		RepoID:        q.RepoID,
		Branch:        q.Branch,
		Version:       q.Version,
		Created:       q.Created,
		Updated:       q.Updated,
		OrderSequence: q.OrderSequence,
	}
}
