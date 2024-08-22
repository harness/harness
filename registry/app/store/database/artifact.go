//  Copyright 2023 Harness, Inc.
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
	"sort"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/app/store/database/util"
	"github.com/harness/gitness/registry/types"
	gitness_store "github.com/harness/gitness/store"
	databaseg "github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type ArtifactDao struct {
	db *sqlx.DB
}

func NewArtifactDao(db *sqlx.DB) store.ArtifactRepository {
	return &ArtifactDao{
		db: db,
	}
}

type artifactDB struct {
	ID         int64          `db:"artifact_id"`
	Name       string         `db:"artifact_name"`
	RegistryID int64          `db:"artifact_registry_id"`
	Labels     sql.NullString `db:"artifact_labels"`
	Enabled    bool           `db:"artifact_enabled"`
	CreatedAt  int64          `db:"artifact_created_at"`
	UpdatedAt  int64          `db:"artifact_updated_at"`
	CreatedBy  int64          `db:"artifact_created_by"`
	UpdatedBy  int64          `db:"artifact_updated_by"`
}

type artifactLabelDB struct {
	Labels sql.NullString `db:"labels"`
}

func (a ArtifactDao) Get(ctx context.Context, id int64) (*types.Artifact, error) {
	q := databaseg.Builder.Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(artifactDB{}), ",")).
		From("artifacts").
		Where("artifact_id = ?", id)

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	dst := new(artifactDB)
	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get artifact")
	}
	return a.mapToArtifact(ctx, dst)
}

func (a ArtifactDao) GetByRepoAndName(ctx context.Context, parentID int64,
	repo string, name string) (*types.Artifact, error) {
	q := databaseg.Builder.Select("a.artifact_id, a.artifact_name, "+
		" a.artifact_registry_id, a.artifact_labels, a.artifact_created_at, "+
		" a.artifact_updated_at, a.artifact_created_by, a.artifact_updated_by").
		From("artifacts a").
		Join(" registries r ON r.registry_id = a.artifact_registry_id").
		Where("r.registry_parent_id = ? AND r.registry_name = ? AND a.artifact_name = ?",
			parentID, repo, name)

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	dst := new(artifactDB)
	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get artifact")
	}
	return a.mapToArtifact(ctx, dst)
}

func (a ArtifactDao) GetByName(ctx context.Context, repoID int64, name string) (*types.Artifact, error) {
	q := databaseg.Builder.Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(artifactDB{}), ",")).
		From("artifacts").
		Where("artifact_registry_id = ? AND artifact_name = ?", repoID, name)

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	dst := new(artifactDB)
	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get artifact")
	}
	return a.mapToArtifact(ctx, dst)
}

func (a ArtifactDao) GetLabelsByParentIDAndRepo(ctx context.Context, parentID int64, repo string,
	limit int, offset int, search string) (labels []string, err error) {
	q := databaseg.Builder.Select("a.artifact_labels as labels").
		From("artifacts a").
		Join("registries r ON r.registry_id = a.artifact_registry_id").
		Where("r.registry_parent_id = ? AND r.registry_name = ?", parentID, repo)

	if search != "" {
		q = q.Where("a.artifact_labels LIKE ?", "%"+search+"%")
	}

	q = q.OrderBy("a.artifact_labels ASC").Limit(uint64(limit)).Offset(uint64(offset))

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	dst := []*artifactLabelDB{}

	db := dbtx.GetAccessor(ctx, a.db)

	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get artifact labels")
	}

	return a.mapToArtifactLabels(dst), nil
}

func (a ArtifactDao) CountLabelsByParentIDAndRepo(ctx context.Context, parentID int64, repo,
	search string) (count int64, err error) {
	q := databaseg.Builder.Select("a.artifact_labels as labels").
		From("artifacts a").
		Join("registries r ON r.registry_id = a.artifact_registry_id").
		Where("r.registry_parent_id = ? AND r.registry_name = ?", parentID, repo)

	if search != "" {
		q = q.Where("a.artifact_labels LIKE ?", "%"+search+"%")
	}

	sql, args, err := q.ToSql()
	if err != nil {
		return -1, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	dst := []*artifactLabelDB{}

	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return -1, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get artifact labels")
	}

	return int64(len(dst)), nil
}

func (a ArtifactDao) GetLabelsByParentID(ctx context.Context, parentID int64) (labels []string, err error) {
	q := databaseg.Builder.Select("a.artifact_labels as labels").
		From("artifacts a").
		Join("registries r ON r.registry_id = a.artifact_registry_id").
		Where("r.registry_parent_id = ?", parentID)

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, a.db)

	dst := []*artifactLabelDB{}

	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get artifact labels")
	}

	return a.mapToArtifactLabels(dst), nil
}

func (a ArtifactDao) CreateOrUpdate(ctx context.Context, artifact *types.Artifact) error {
	const sqlQuery = `
		INSERT INTO artifacts ( 
		         artifact_registry_id
				,artifact_name
				,artifact_enabled
				,artifact_created_at
				,artifact_updated_at
				,artifact_created_by
				,artifact_updated_by
		    ) VALUES (
						 :artifact_registry_id
						,:artifact_name
						,:artifact_enabled
						,:artifact_created_at
						,:artifact_updated_at
						,:artifact_created_by
						,:artifact_updated_by
		    ) 
            ON CONFLICT (artifact_registry_id, artifact_name)
		    DO UPDATE SET
			   artifact_enabled = :artifact_enabled
            RETURNING artifact_id`

	db := dbtx.GetAccessor(ctx, a.db)
	query, arg, err := db.BindNamed(sqlQuery, a.mapToInternalArtifact(ctx, artifact))
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Failed to bind artifact object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&artifact.ID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return databaseg.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}
	return nil
}

func (a ArtifactDao) Update(ctx context.Context, artifact *types.Artifact) (err error) {
	var sqlQuery = " UPDATE artifacts SET " + util.GetSetDBKeys(artifactDB{}, "artifact_id") +
		" WHERE artifact_id = :artifact_id "

	dbArtifact := a.mapToInternalArtifact(ctx, artifact)

	// update Version (used for optimistic locking) and Updated time
	dbArtifact.UpdatedAt = time.Now().UnixMilli()

	db := dbtx.GetAccessor(ctx, a.db)

	query, arg, err := db.BindNamed(sqlQuery, dbArtifact)
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Failed to bind artifact object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Failed to update artifact")
	}

	count, err := result.RowsAffected()
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Failed to get number of updated rows")
	}

	if count == 0 {
		return gitness_store.ErrVersionConflict
	}

	return nil
}

func (a ArtifactDao) mapToInternalArtifact(ctx context.Context, in *types.Artifact) *artifactDB {
	session, _ := request.AuthSessionFrom(ctx)

	if in.CreatedAt.IsZero() {
		in.CreatedAt = time.Now()
	}
	if in.CreatedBy == 0 {
		in.CreatedBy = session.Principal.ID
	}

	in.UpdatedAt = time.Now()
	in.UpdatedBy = session.Principal.ID

	sort.Strings(in.Labels)

	return &artifactDB{
		ID:         in.ID,
		Name:       in.Name,
		RegistryID: in.RegistryID,
		Labels:     getEmptySQLString(util.ArrToString(in.Labels)),
		Enabled:    in.Enabled,
		CreatedAt:  in.CreatedAt.UnixMilli(),
		UpdatedAt:  in.UpdatedAt.UnixMilli(),
		CreatedBy:  in.CreatedBy,
		UpdatedBy:  in.UpdatedBy,
	}
}

func (a ArtifactDao) mapToArtifact(_ context.Context, dst *artifactDB) (*types.Artifact, error) {
	createdBy := dst.CreatedBy
	updatedBy := dst.UpdatedBy
	return &types.Artifact{
		ID:         dst.ID,
		Name:       dst.Name,
		RegistryID: dst.RegistryID,
		Labels:     util.StringToArr(dst.Labels.String),
		Enabled:    dst.Enabled,
		CreatedAt:  time.UnixMilli(dst.CreatedAt),
		UpdatedAt:  time.UnixMilli(dst.UpdatedAt),
		CreatedBy:  createdBy,
		UpdatedBy:  updatedBy,
	}, nil
}

func (a ArtifactDao) mapToArtifactLabels(dst []*artifactLabelDB) []string {
	elements := make(map[string]bool)
	res := []string{}
	for _, labels := range dst {
		elements, res = a.mapToArtifactLabel(elements, res, labels)
	}
	return res
}

func (a ArtifactDao) mapToArtifactLabel(elements map[string]bool, res []string,
	dst *artifactLabelDB) (map[string]bool, []string) {
	if dst == nil {
		return elements, res
	}
	labels := util.StringToArr(dst.Labels.String)
	for _, label := range labels {
		if !elements[label] {
			elements[label] = true
			res = append(res, label)
		}
	}
	return elements, res
}
