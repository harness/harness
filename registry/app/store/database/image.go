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
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/app/store/database/util"
	"github.com/harness/gitness/registry/types"
	gitness_store "github.com/harness/gitness/store"
	databaseg "github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type ImageDao struct {
	db *sqlx.DB
}

func NewImageDao(db *sqlx.DB) store.ImageRepository {
	return &ImageDao{
		db: db,
	}
}

type imageDB struct {
	ID           int64                  `db:"image_id"`
	UUID         string                 `db:"image_uuid"`
	Name         string                 `db:"image_name"`
	ArtifactType *artifact.ArtifactType `db:"image_type"`
	RegistryID   int64                  `db:"image_registry_id"`
	Labels       sql.NullString         `db:"image_labels"`
	Enabled      bool                   `db:"image_enabled"`
	CreatedAt    int64                  `db:"image_created_at"`
	UpdatedAt    int64                  `db:"image_updated_at"`
	CreatedBy    int64                  `db:"image_created_by"`
	UpdatedBy    int64                  `db:"image_updated_by"`
}

type imageLabelDB struct {
	Labels sql.NullString `db:"labels"`
}

func (i ImageDao) Get(ctx context.Context, id int64) (*types.Image, error) {
	q := databaseg.Builder.Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(imageDB{}), ",")).
		From("images").
		Where("image_id = ?", id)

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, i.db)

	dst := new(imageDB)
	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get image")
	}
	return i.mapToImage(ctx, dst)
}

func (i ImageDao) DeleteByImageNameAndRegID(ctx context.Context, regID int64, image string) (err error) {
	stmt := databaseg.Builder.Delete("images").
		Where("image_name = ? AND image_registry_id = ?", image, regID)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, i.db)

	_, err = db.ExecContext(ctx, sql, args...)
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "the delete query failed")
	}

	return nil
}

func (i ImageDao) DeleteByImageNameIfNoLinkedArtifacts(
	ctx context.Context, regID int64, image string,
) error {
	stmt := databaseg.Builder.Delete("images").
		Where("image_name = ? AND image_registry_id = ?", image, regID).
		Where("NOT EXISTS ( SELECT 1 FROM artifacts WHERE artifacts.artifact_image_id = images.image_id )")

	sql, args, err := stmt.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, i.db)

	_, err = db.ExecContext(ctx, sql, args...)
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "the delete query failed")
	}

	return nil
}

func (i ImageDao) GetByName(ctx context.Context, registryID int64, name string) (*types.Image, error) {
	q := databaseg.Builder.Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(imageDB{}), ",")).
		From("images").
		Where("image_registry_id = ? AND image_name = ?", registryID, name)

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, i.db)

	dst := new(imageDB)
	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get image")
	}
	return i.mapToImage(ctx, dst)
}
func (i ImageDao) GetByNameAndType(ctx context.Context, registryID int64, name string,
	artifactType *artifact.ArtifactType) (*types.Image, error) {
	q := databaseg.Builder.Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(imageDB{}), ",")).
		From("images").
		Where("image_registry_id = ? AND image_name = ?", registryID, name)

	if artifactType != nil && *artifactType != "" {
		q = q.Where("image_type = ?", *artifactType)
	}

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, i.db)

	dst := new(imageDB)
	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get image")
	}
	return i.mapToImage(ctx, dst)
}

func (i ImageDao) CreateOrUpdate(ctx context.Context, image *types.Image) error {
	if commons.IsEmpty(image.Name) {
		return errors.New("package/image name is empty")
	}
	var conflictCondition string
	if image.ArtifactType == nil {
		conflictCondition = ` ON CONFLICT (image_registry_id, image_name) WHERE image_type IS NULL `
	} else {
		conflictCondition = ` ON CONFLICT (image_registry_id, image_name, image_type) WHERE image_type IS NOT NULL `
	}
	var sqlQuery = `
		INSERT INTO images ( 
		         image_registry_id
				,image_name
				,image_type
				,image_enabled
				,image_created_at
				,image_updated_at
				,image_created_by
				,image_updated_by
				,image_uuid
		    ) VALUES (
						 :image_registry_id
						,:image_name
						,:image_type
						,:image_enabled
						,:image_created_at
						,:image_updated_at
						,:image_created_by
						,:image_updated_by
					    ,:image_uuid
		    ) 
           ` + conflictCondition + `
		    DO UPDATE SET
			   image_enabled = :image_enabled
            RETURNING image_id`

	db := dbtx.GetAccessor(ctx, i.db)
	query, arg, err := db.BindNamed(sqlQuery, i.mapToInternalImage(ctx, image))
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Failed to bind image object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&image.ID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return databaseg.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}
	return nil
}

func (i ImageDao) GetLabelsByParentIDAndRepo(
	ctx context.Context, parentID int64, repo string,
	limit int, offset int, search string,
) (labels []string, err error) {
	q := databaseg.Builder.Select("a.image_labels as labels").
		From("images a").
		Join("registries r ON r.registry_id = a.image_registry_id").
		Where("r.registry_parent_id = ? AND r.registry_name = ?", parentID, repo)

	if search != "" {
		q = q.Where("a.image_labels LIKE ?", "%"+search+"%")
	}

	q = q.OrderBy("a.image_labels ASC").
		Limit(util.SafeIntToUInt64(limit)).Offset(util.SafeIntToUInt64(offset))

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	dst := []*imageLabelDB{}

	db := dbtx.GetAccessor(ctx, i.db)

	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get artifact labels")
	}

	return i.mapToImageLabels(dst), nil
}

func (i ImageDao) CountLabelsByParentIDAndRepo(
	ctx context.Context, parentID int64, repo,
	search string,
) (count int64, err error) {
	q := databaseg.Builder.Select("a.image_labels as labels").
		From("images a").
		Join("registries r ON r.registry_id = a.image_registry_id").
		Where("r.registry_parent_id = ? AND r.registry_name = ?", parentID, repo)

	if search != "" {
		q = q.Where("a.image_labels LIKE ?", "%"+search+"%")
	}

	sql, args, err := q.ToSql()
	if err != nil {
		return -1, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, i.db)

	dst := []*imageLabelDB{}

	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return -1, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get artifact labels")
	}

	return int64(len(dst)), nil
}

func (i ImageDao) GetByRepoAndName(
	ctx context.Context, parentID int64,
	repo string, name string,
) (*types.Image, error) {
	q := databaseg.Builder.Select("a.image_id, a.image_name, "+
		" a.image_registry_id, a.image_labels, a.image_created_at, "+
		" a.image_updated_at, a.image_created_by, a.image_updated_by").
		From("images a").
		Join(" registries r ON r.registry_id = a.image_registry_id").
		Where("r.registry_parent_id = ? AND r.registry_name = ? AND a.image_name = ?",
			parentID, repo, name)

	sql, args, err := q.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	db := dbtx.GetAccessor(ctx, i.db)

	dst := new(imageDB)
	if err = db.GetContext(ctx, dst, sql, args...); err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to get artifact")
	}
	return i.mapToImage(ctx, dst)
}

func (i ImageDao) Update(ctx context.Context, image *types.Image) (err error) {
	var sqlQuery = " UPDATE images SET " + util.GetSetDBKeys(imageDB{}, "image_id") +
		" WHERE image_id = :image_id "

	dbImage := i.mapToInternalImage(ctx, image)

	// update Version (used for optimistic locking) and Updated time
	dbImage.UpdatedAt = time.Now().UnixMilli()

	db := dbtx.GetAccessor(ctx, i.db)

	query, arg, err := db.BindNamed(sqlQuery, dbImage)
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Failed to bind images object")
	}

	result, err := db.ExecContext(ctx, query, arg...)
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Failed to update images")
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

func (i ImageDao) UpdateStatus(ctx context.Context, image *types.Image) (err error) {
	q := databaseg.Builder.Update("images").
		Set("image_enabled", image.Enabled).
		Set("image_updated_at", time.Now().UnixMilli()).
		Where("image_registry_id = ? AND image_name = ?",
			image.RegistryID, image.Name)

	sql, args, err := q.ToSql()

	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Failed to bind images object")
	}

	result, err := i.db.ExecContext(ctx, sql, args...)
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Failed to update images")
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

func (i ImageDao) mapToInternalImage(ctx context.Context, in *types.Image) *imageDB {
	session, _ := request.AuthSessionFrom(ctx)

	if in.CreatedAt.IsZero() {
		in.CreatedAt = time.Now()
	}
	if in.CreatedBy == 0 {
		in.CreatedBy = session.Principal.ID
	}

	in.UpdatedAt = time.Now()
	in.UpdatedBy = session.Principal.ID

	if in.UUID == "" {
		in.UUID = uuid.NewString()
	}

	sort.Strings(in.Labels)

	return &imageDB{
		ID:           in.ID,
		UUID:         in.UUID,
		Name:         in.Name,
		ArtifactType: in.ArtifactType,
		RegistryID:   in.RegistryID,
		Labels:       util.GetEmptySQLString(util.ArrToString(in.Labels)),
		Enabled:      in.Enabled,
		CreatedAt:    in.CreatedAt.UnixMilli(),
		UpdatedAt:    in.UpdatedAt.UnixMilli(),
		CreatedBy:    in.CreatedBy,
		UpdatedBy:    in.UpdatedBy,
	}
}

func (i ImageDao) mapToImage(_ context.Context, dst *imageDB) (*types.Image, error) {
	createdBy := dst.CreatedBy
	updatedBy := dst.UpdatedBy
	return &types.Image{
		ID:           dst.ID,
		UUID:         dst.UUID,
		Name:         dst.Name,
		ArtifactType: dst.ArtifactType,
		RegistryID:   dst.RegistryID,
		Labels:       util.StringToArr(dst.Labels.String),
		Enabled:      dst.Enabled,
		CreatedAt:    time.UnixMilli(dst.CreatedAt),
		UpdatedAt:    time.UnixMilli(dst.UpdatedAt),
		CreatedBy:    createdBy,
		UpdatedBy:    updatedBy,
	}, nil
}

func (i ImageDao) mapToImageLabels(dst []*imageLabelDB) []string {
	elements := make(map[string]bool)
	res := []string{}
	for _, labels := range dst {
		elements, res = i.mapToImageLabel(elements, res, labels)
	}
	return res
}

func (i ImageDao) mapToImageLabel(
	elements map[string]bool, res []string,
	dst *imageLabelDB,
) (map[string]bool, []string) {
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
