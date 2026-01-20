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

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
)

var _ store.AutoMergeStore = (*AutoMergeStore)(nil)

func NewAutoMergeStore(db *sqlx.DB) *AutoMergeStore {
	return &AutoMergeStore{
		db: db,
	}
}

type AutoMergeStore struct {
	db *sqlx.DB
}

type autoMergeInternal struct {
	PullReqID    int64            `db:"auto_merge_pullreq_id"`
	Requested    int64            `db:"auto_merge_requested"`
	RequestedBy  int64            `db:"auto_merge_requested_by"`
	MergeMethod  enum.MergeMethod `db:"auto_merge_method"`
	Title        string           `db:"auto_merge_title"`
	Message      string           `db:"auto_merge_message"`
	DeleteBranch bool             `db:"auto_merge_delete_branch"`
}

const autoMergeColumns = `
		 auto_merge_pullreq_id
		,auto_merge_requested
		,auto_merge_requested_by
		,auto_merge_method
		,auto_merge_title
		,auto_merge_message
		,auto_merge_delete_branch`

func (s *AutoMergeStore) Find(ctx context.Context, pullreqID int64) (*types.AutoMerge, error) {
	stmt := database.Builder.
		Select(autoMergeColumns).
		From("auto_merges").
		Where("auto_merge_pullreq_id = ?", pullreqID)

	sqlQuery, args, err := stmt.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to convert delete auto merge query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	var result autoMergeInternal
	err = db.GetContext(ctx, &result, sqlQuery, args...)
	if err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to find auto merge by uid")
	}

	return (*types.AutoMerge)(&result), nil
}

func (s *AutoMergeStore) Delete(ctx context.Context, pullreqID int64) (bool, error) {
	stmt := database.Builder.
		Delete("auto_merges").
		Where("auto_merge_pullreq_id = ?", pullreqID)

	sqlQuery, args, err := stmt.ToSql()
	if err != nil {
		return false, fmt.Errorf("failed to convert delete auto merge query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)

	result, err := db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return false, database.ProcessSQLErrorf(ctx, err, "failed to execute delete auto merge query")
	}

	n, err := result.RowsAffected()
	if err != nil {
		return false, database.ProcessSQLErrorf(ctx, err, "failed to get number of auto merge deleted rows")
	}

	return n > 0, nil
}

func (s *AutoMergeStore) Upsert(ctx context.Context, autoMerge *types.AutoMerge) error {
	const sqlQuery = `
		INSERT INTO auto_merges (` + autoMergeColumns + `
		) VALUES (
			 :auto_merge_pullreq_id
			,:auto_merge_requested
			,:auto_merge_requested_by
			,:auto_merge_method
			,:auto_merge_title
			,:auto_merge_message
			,:auto_merge_delete_branch
		)
		ON CONFLICT (auto_merge_pullreq_id) DO
		UPDATE SET
			 auto_merge_requested = :auto_merge_requested
			,auto_merge_requested_by = :auto_merge_requested_by
			,auto_merge_method = :auto_merge_method
			,auto_merge_title = :auto_merge_title
			,auto_merge_message = :auto_merge_message
			,auto_merge_delete_branch = :auto_merge_delete_branch`

	db := dbtx.GetAccessor(ctx, s.db)

	query, arg, err := db.BindNamed(sqlQuery, (*autoMergeInternal)(autoMerge))
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "failed to bind auto merge object")
	}

	_, err = db.ExecContext(ctx, query, arg...)
	if err != nil {
		return database.ProcessSQLErrorf(ctx, err, "upsert auto merge query failed")
	}

	return nil
}
