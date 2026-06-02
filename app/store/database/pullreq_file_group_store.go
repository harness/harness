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
	"database/sql"

	appstore "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var _ appstore.PullReqFileGroupStore = (*PullReqFileGroupStore)(nil)

// NewPullReqFileGroupStore returns a new PullReqFileGroupStore.
func NewPullReqFileGroupStore(db *sqlx.DB) *PullReqFileGroupStore {
	return &PullReqFileGroupStore{db: db}
}

// PullReqFileGroupStore implements store.PullReqFileGroupStore backed by a relational database.
type PullReqFileGroupStore struct {
	db *sqlx.DB
}

type pullReqFileGroupRow struct {
	groupID          int64
	groupPullReqID   int64
	groupTitle       string
	groupDescription string
	groupCreated     int64
	groupUpdated     int64
	groupCreatedBy   int64
	groupUpdatedBy   int64
	filePath         sql.NullString
	fileOldSHA       sql.NullString
	fileNewSHA       sql.NullString
}

// DeleteByPrID deletes all pull request file groups for the PR.
func (s *PullReqFileGroupStore) DeleteByPrID(ctx context.Context, prID int64) error {
	db := dbtx.GetAccessor(ctx, s.db)

	deleteStmt := database.Builder.
		Delete("pullreq_file_groups").
		Where(squirrel.Eq{"pullreq_file_group_pr_id": prID})

	deleteQuery, deleteArgs, err := deleteStmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "failed to convert delete pull request file groups query to sql")
	}

	if _, err = db.ExecContext(ctx, deleteQuery, deleteArgs...); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Delete pull request file groups query failed")
	}

	return nil
}

// CreateMany inserts the provided pull request file groups for the PR.
func (s *PullReqFileGroupStore) CreateMany(
	ctx context.Context,
	groups []*types.PullReqFileGroupWithFiles,
) error {
	db := dbtx.GetAccessor(ctx, s.db)

	for _, group := range groups {
		insertGroupStmt := database.Builder.
			Insert("pullreq_file_groups").
			Columns(
				"pullreq_file_group_pr_id",
				"pullreq_file_group_title",
				"pullreq_file_group_description",
				"pullreq_file_group_created",
				"pullreq_file_group_updated",
				"pullreq_file_group_created_by",
				"pullreq_file_group_updated_by",
			).
			Values(
				group.PullReqID,
				group.Title,
				group.Description,
				group.Created,
				group.Updated,
				group.CreatedBy,
				group.UpdatedBy,
			).
			Suffix(ReturningClause + "pullreq_file_group_id")

		insertGroupQuery, insertGroupArgs, err := insertGroupStmt.ToSql()
		if err != nil {
			return errors.Wrap(err, "failed to convert create pull request file group query to sql")
		}

		if err = db.QueryRowContext(ctx, insertGroupQuery, insertGroupArgs...).Scan(&group.ID); err != nil {
			return database.ProcessSQLErrorf(ctx, err, "Create pull request file group query failed")
		}

		if len(group.Files) == 0 {
			continue
		}

		insertFilesStmt := database.Builder.
			Insert("pullreq_file_group_files").
			Columns(
				"pullreq_file_group_file_pullreq_file_group_id",
				"pullreq_file_group_file_pr_id",
				"pullreq_file_group_file_path",
				"pullreq_file_group_file_old_sha",
				"pullreq_file_group_file_new_sha",
			)

		for _, file := range group.Files {
			insertFilesStmt = insertFilesStmt.Values(
				group.ID,
				group.PullReqID,
				file.Path,
				normalizeNullableSHA(file.OldSHA),
				normalizeNullableSHA(file.NewSHA),
			)
		}

		insertFilesQuery, insertFilesArgs, err := insertFilesStmt.ToSql()
		if err != nil {
			return errors.Wrap(err, "failed to convert create pull request file group files query to sql")
		}

		if _, err = db.ExecContext(ctx, insertFilesQuery, insertFilesArgs...); err != nil {
			return database.ProcessSQLErrorf(ctx, err, "Create pull request file group files query failed")
		}
	}

	return nil
}

// List returns all pull request file groups with their files.
func (s *PullReqFileGroupStore) List(ctx context.Context, prID int64) ([]*types.PullReqFileGroupWithFiles, error) {
	db := dbtx.GetAccessor(ctx, s.db)

	stmt := database.Builder.
		Select(
			"g.pullreq_file_group_id",
			"g.pullreq_file_group_pr_id",
			"g.pullreq_file_group_title",
			"g.pullreq_file_group_description",
			"g.pullreq_file_group_created",
			"g.pullreq_file_group_updated",
			"g.pullreq_file_group_created_by",
			"g.pullreq_file_group_updated_by",
			"f.pullreq_file_group_file_path",
			"f.pullreq_file_group_file_old_sha",
			"f.pullreq_file_group_file_new_sha",
		).
		From("pullreq_file_groups g").
		LeftJoin("pullreq_file_group_files f ON f.pullreq_file_group_file_pullreq_file_group_id = g.pullreq_file_group_id").
		Where(squirrel.Eq{"g.pullreq_file_group_pr_id": prID}).
		OrderBy("g.pullreq_file_group_id", "f.pullreq_file_group_file_path")

	query, args, err := stmt.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert list pull request file groups query to sql")
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "List pull request file groups query failed")
	}
	defer func() {
		_ = rows.Close()
	}()

	groupsByID := make(map[int64]*types.PullReqFileGroupWithFiles)
	groupOrder := make([]int64, 0)

	for rows.Next() {
		row := pullReqFileGroupRow{}
		if err = rows.Scan(
			&row.groupID,
			&row.groupPullReqID,
			&row.groupTitle,
			&row.groupDescription,
			&row.groupCreated,
			&row.groupUpdated,
			&row.groupCreatedBy,
			&row.groupUpdatedBy,
			&row.filePath,
			&row.fileOldSHA,
			&row.fileNewSHA,
		); err != nil {
			return nil, database.ProcessSQLErrorf(ctx, err, "failed to scan pull request file groups")
		}

		group, ok := groupsByID[row.groupID]
		if !ok {
			group = &types.PullReqFileGroupWithFiles{
				PullReqFileGroup: types.PullReqFileGroup{
					ID:          row.groupID,
					PullReqID:   row.groupPullReqID,
					Title:       row.groupTitle,
					Description: row.groupDescription,
					Created:     row.groupCreated,
					Updated:     row.groupUpdated,
					CreatedBy:   row.groupCreatedBy,
					UpdatedBy:   row.groupUpdatedBy,
				},
				Files: make([]*types.PullReqFileGroupFile, 0),
			}
			groupsByID[row.groupID] = group
			groupOrder = append(groupOrder, row.groupID)
		}

		if row.filePath.Valid {
			group.Files = append(group.Files, &types.PullReqFileGroupFile{
				PullReqFileGroupID: row.groupID,
				Path:               row.filePath.String,
				OldSHA:             row.fileOldSHA.String,
				NewSHA:             row.fileNewSHA.String,
			})
		}
	}

	if err = rows.Err(); err != nil {
		return nil, database.ProcessSQLErrorf(ctx, err, "failed to read pull request file groups data")
	}

	result := make([]*types.PullReqFileGroupWithFiles, 0, len(groupOrder))
	for _, groupID := range groupOrder {
		result = append(result, groupsByID[groupID])
	}

	return result, nil
}

func normalizeNullableSHA(sha string) any {
	if sha == "" || sha == types.NilSHA {
		return nil
	}

	return sha
}
