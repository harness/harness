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
	"fmt"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/types"
	databaseg "github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type PackageTagDao struct {
	db *sqlx.DB
}

func NewPackageTagDao(db *sqlx.DB) *PackageTagDao {
	return &PackageTagDao{
		db: db,
	}
}

type PackageTagDB struct {
	ID         string `db:"package_tag_id"`
	Name       string `db:"package_tag_name"`
	ArtifactID int64  `db:"package_tag_artifact_id"`
	CreatedAt  int64  `db:"package_tag_created_at"`
	CreatedBy  int64  `db:"package_tag_created_by"`
	UpdatedAt  int64  `db:"package_tag_updated_at"`
	UpdatedBy  int64  `db:"package_tag_updated_by"`
}

type PackageTagMetadataDB struct {
	ID      string `db:"package_tag_id"`
	Name    string `db:"package_tag_name"`
	ImageID int64  `db:"package_tag_image_id"`
	Version string `db:"package_tag_version"`
}

func (r PackageTagDao) FindByImageNameAndRegID(ctx context.Context,
	image string, regID int64) ([]*types.PackageTagMetadata, error) {
	stmt := databaseg.Builder.
		Select("p.package_tag_id as package_tag_id, "+
			"p.package_tag_name as package_tag_name,"+
			"i.image_id as package_tag_image_id,"+
			"a.artifact_version as package_tag_version").
		From("package_tags as p").
		Join("artifacts as a ON p.package_tag_artifact_id = a.artifact_id").
		Join("images as i ON a.artifact_image_id = i.image_id").
		Join("registries as r ON i.image_registry_id = r.registry_id").
		Where("i.image_name = ? AND r.registry_id = ?", image, regID)

	db := dbtx.GetAccessor(ctx, r.db)

	dst := []PackageTagMetadataDB{}

	sql, args, err := stmt.ToSql()
	if err != nil {
		return nil,
			errors.Wrap(err, "Failed to convert query to sql")
	}
	if err = db.SelectContext(ctx, &dst, sql, args...); err != nil {
		return nil,
			databaseg.ProcessSQLErrorf(ctx, err, "Failed to find package tags")
	}

	return r.mapToPackageTagList(ctx, dst)
}

// DeleteByTagAndImageName Todo:postgres query can be optimised here
func (r PackageTagDao) DeleteByTagAndImageName(ctx context.Context,
	tag string, image string, regID int64) error {
	stmt := databaseg.Builder.Delete("package_tags").
		Where("package_tag_id IN (SELECT p.package_tag_id FROM package_tags p"+
			" JOIN artifacts a ON p.package_tag_artifact_id = a.artifact_id "+
			"JOIN images i ON a.artifact_image_id = i.image_id "+
			" JOIN registries r ON r.registry_id = i.image_registry_id  "+
			"WHERE i.image_name = ? AND p.package_tag_name = ? AND r.registry_id = ?)", image, tag, regID)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return fmt.Errorf("failed to convert purge package_tag query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, r.db)

	_, err = db.ExecContext(ctx, sql, args...)
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "the delete query failed")
	}

	return nil
}

func (r PackageTagDao) DeleteByImageNameAndRegID(ctx context.Context, image string, regID int64) error {
	stmt := databaseg.Builder.Delete("package_tags").
		Where("package_tag_id IN (SELECT p.package_tag_id FROM package_tags p "+
			"JOIN artifacts a ON p.package_tag_artifact_id = a.artifact_id "+
			"JOIN images i ON a.artifact_image_id = i.image_id "+
			" JOIN registries r ON r.registry_id = i.image_registry_id  "+
			"WHERE i.image_name = ? AND r.registry_id = ?)", image, regID)

	sql, args, err := stmt.ToSql()
	if err != nil {
		return fmt.Errorf("failed to convert purge package_tag query to sql: %w", err)
	}

	db := dbtx.GetAccessor(ctx, r.db)

	_, err = db.ExecContext(ctx, sql, args...)
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "the delete query failed")
	}

	return nil
}

func (r PackageTagDao) Create(ctx context.Context, tag *types.PackageTag) (string, error) {
	const sqlQuery = `
		INSERT INTO package_tags ( 
			package_tag_id,
			package_tag_name,
			package_tag_artifact_id,
			package_tag_created_at,
			package_tag_created_by,
			package_tag_updated_at,
			package_tag_updated_by
		) VALUES (
		          :package_tag_id,
			:package_tag_name,
			:package_tag_artifact_id,
			:package_tag_created_at,
			:package_tag_created_by,
			:package_tag_updated_at,
			:package_tag_updated_by
		) 
			ON CONFLICT (package_tag_name, 
			package_tag_artifact_id)
	    DO UPDATE SET
	    package_tag_artifact_id = :package_tag_artifact_id 
		RETURNING package_tag_id`

	db := dbtx.GetAccessor(ctx, r.db)
	query, arg, err := db.BindNamed(sqlQuery, mapToInternalPackageTag(ctx, tag))
	if err != nil {
		return "",
			databaseg.ProcessSQLErrorf(ctx, err, "Failed to bind repo object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&tag.ID); err != nil {
		return "", databaseg.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}

	return tag.ID, nil
}

func (r PackageTagDao) mapToPackageTagList(ctx context.Context,
	dst []PackageTagMetadataDB) ([]*types.PackageTagMetadata, error) {
	tags := make([]*types.PackageTagMetadata, 0, len(dst))
	for _, d := range dst {
		tag := r.mapToPackageTag(ctx, d)
		tags = append(tags, tag)
	}
	return tags, nil
}
func (r PackageTagDao) mapToPackageTag(_ context.Context, dst PackageTagMetadataDB) *types.PackageTagMetadata {
	return &types.PackageTagMetadata{
		ID:      dst.ID,
		Name:    dst.Name,
		Version: dst.Version}
}

func mapToInternalPackageTag(ctx context.Context, in *types.PackageTag) *PackageTagDB {
	session, _ := request.AuthSessionFrom(ctx)
	if in.CreatedAt.IsZero() {
		in.CreatedAt = time.Now()
	}
	in.UpdatedAt = time.Now()
	if in.CreatedBy == 0 {
		in.CreatedBy = session.Principal.ID
	}
	in.UpdatedBy = session.Principal.ID

	return &PackageTagDB{
		ID:         in.ID,
		Name:       in.Name,
		ArtifactID: in.ArtifactID,
		CreatedAt:  in.CreatedAt.UnixMilli(),
		UpdatedAt:  in.UpdatedAt.UnixMilli(),
		CreatedBy:  in.CreatedBy,
		UpdatedBy:  in.UpdatedBy,
	}
}
