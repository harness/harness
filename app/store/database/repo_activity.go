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
	"encoding/json"
	"fmt"

	"github.com/harness/gitness/app/store"
	storedb "github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

var _ store.RepoActivityStore = (*RepoActivityStore)(nil)

// RepoActivityStore implements store.RepoActivityStore backed by a relational database.
type RepoActivityStore struct {
	db *sqlx.DB
}

func NewRepoActivityStore(db *sqlx.DB) *RepoActivityStore {
	return &RepoActivityStore{db: db}
}

type repoActivity struct {
	Key string `db:"repo_activity_key"`

	RepoID      int64 `db:"repo_activity_repo_id"`
	PrincipalID int64 `db:"repo_activity_principal_id"`

	Type      string          `db:"repo_activity_type"`
	Payload   json.RawMessage `db:"repo_activity_payload"`
	CreatedAt int64           `db:"repo_activity_created_at"`
}

const repoActivityColumns = `
	repo_activity_key
	,repo_activity_repo_id
	,repo_activity_principal_id
	,repo_activity_type
	,repo_activity_payload
	,repo_activity_created_at`

func (s *RepoActivityStore) Create(ctx context.Context, activity *types.RepoActivity) error {
	const sqlQuery = `
	INSERT INTO repo_activities (
		 repo_activity_key
		,repo_activity_repo_id
		,repo_activity_principal_id
		,repo_activity_type
		,repo_activity_payload
		,repo_activity_created_at
	) VALUES (
		 :repo_activity_key
		,:repo_activity_repo_id
		,:repo_activity_principal_id
		,:repo_activity_type
		,:repo_activity_payload
		,:repo_activity_created_at
	)
	ON CONFLICT (repo_activity_key) DO NOTHING`

	db := dbtx.GetAccessor(ctx, s.db)
	dbActivity, err := mapToRepoActivity(activity)
	if err != nil {
		return fmt.Errorf("failed to map repository activity: %w", err)
	}

	query, args, err := db.BindNamed(sqlQuery, dbActivity)
	if err != nil {
		return storedb.ProcessSQLErrorf(ctx, err, "Failed to bind repository activity object")
	}

	_, err = db.ExecContext(ctx, query, args...)
	if err != nil {
		return storedb.ProcessSQLErrorf(ctx, err, "Failed to insert repository activity")
	}

	return nil
}

func (s *RepoActivityStore) List(
	ctx context.Context,
	repoID int64,
	filter *types.RepoActivityFilter,
) ([]*types.RepoActivity, error) {
	stmt := s.applyRepoActivityFilter(
		squirrel.Select(repoActivityColumns).From("repo_activities"),
		repoID,
		filter,
	)

	stmt = stmt.OrderBy("repo_activity_created_at ASC").
		Offset(storedb.Offset(filter.Page, filter.Size)).
		Limit(uint64(filter.Size)) //nolint:gosec

	query, args, err := stmt.PlaceholderFormat(squirrel.Question).ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build repo activity list query: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)
	query = db.Rebind(query)

	dst := []*repoActivity{}
	if err = db.SelectContext(ctx, &dst, query, args...); err != nil {
		return nil, storedb.ProcessSQLErrorf(ctx, err, "Failed executing repository activity list query")
	}

	result := make([]*types.RepoActivity, len(dst))
	for i := range dst {
		result[i], err = mapFromRepoActivity(dst[i])
		if err != nil {
			return nil, fmt.Errorf("failed to map repository activity from database: %w", err)
		}
	}

	return result, nil
}

func (s *RepoActivityStore) Count(
	ctx context.Context,
	repoID int64,
	filter *types.RepoActivityFilter,
) (int, error) {
	stmt := s.applyRepoActivityFilter(
		squirrel.Select("COUNT(*)").From("repo_activities"),
		repoID,
		filter,
	)

	query, args, err := stmt.PlaceholderFormat(squirrel.Question).ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to build repo activity count query: %w", err)
	}

	db := dbtx.GetAccessor(ctx, s.db)
	query = db.Rebind(query)

	total := 0
	if err = db.GetContext(ctx, &total, query, args...); err != nil {
		return 0, storedb.ProcessSQLErrorf(ctx, err, "Failed executing repository activity count query")
	}

	return total, nil
}

func (s *RepoActivityStore) applyRepoActivityFilter(
	stmt squirrel.SelectBuilder,
	repoID int64,
	filter *types.RepoActivityFilter,
) squirrel.SelectBuilder {
	stmt = stmt.Where(squirrel.Eq{"repo_activity_repo_id": repoID})

	if filter.After > 0 {
		stmt = stmt.Where(squirrel.Gt{"repo_activity_created_at": filter.After})
	}
	if filter.Before > 0 {
		stmt = stmt.Where(squirrel.Lt{"repo_activity_created_at": filter.Before})
	}

	return stmt
}

func mapToRepoActivity(in *types.RepoActivity) (*repoActivity, error) {
	if in.Payload != nil && in.Payload.ActivityType() != in.Type {
		return nil, fmt.Errorf(
			"repository activity payload type mismatch: activity=%s payload=%s",
			in.Type, in.Payload.ActivityType(),
		)
	}

	data, err := types.MarshalRepoActivityPayload(in.Payload)
	if err != nil {
		return nil, err
	}

	out := &repoActivity{
		Key:         in.Key,
		RepoID:      in.RepoID,
		PrincipalID: in.PrincipalID,
		Type:        string(in.Type),
		Payload:     data,
		CreatedAt:   in.Timestamp,
	}

	return out, nil
}

func mapFromRepoActivity(in *repoActivity) (*types.RepoActivity, error) {
	activityType, err := enum.ParseRepoActivityType(in.Type)
	if err != nil {
		return nil, err
	}

	data, err := types.UnmarshalRepoActivityPayload(activityType, in.Payload)
	if err != nil {
		return nil, err
	}

	out := &types.RepoActivity{
		Key:         in.Key,
		RepoID:      in.RepoID,
		PrincipalID: in.PrincipalID,
		Payload:     data,
		Timestamp:   in.CreatedAt,
	}

	return out, nil
}
