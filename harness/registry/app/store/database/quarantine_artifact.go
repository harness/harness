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
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/store/database/util"
	"github.com/harness/gitness/registry/types"
	databaseg "github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type QuarantineArtifactDao struct {
	db *sqlx.DB
}

func NewQuarantineArtifactDao(db *sqlx.DB) *QuarantineArtifactDao {
	return &QuarantineArtifactDao{
		db: db,
	}
}

type QuarantineArtifactDB struct {
	ID         string  `db:"quarantined_path_id"`
	NodeID     *string `db:"quarantined_path_node_id"`
	Reason     string  `db:"quarantined_path_reason"`
	RegistryID int64   `db:"quarantined_path_registry_id"`
	ImageID    *int64  `db:"quarantined_path_image_id"`
	ArtifactID *int64  `db:"quarantined_path_artifact_id"`
	CreatedAt  int64   `db:"quarantined_path_created_at"`
	CreatedBy  int64   `db:"quarantined_path_created_by"`
}

func (q QuarantineArtifactDao) GetByFilePath(ctx context.Context,
	filePath string, registryID int64,
	artifact string, version string) ([]*types.QuarantineArtifact, error) {
	// First, get all quarantine artifacts for this registry
	stmtBuilder := databaseg.Builder.
		Select(util.ArrToStringByDelimiter(util.GetDBTagsFromStruct(QuarantineArtifactDB{}), ",")).
		From("quarantined_paths").
		LeftJoin("artifacts as ar ON quarantined_path_artifact_id = ar.artifact_id").
		LeftJoin("images as i ON quarantined_path_image_id = i.image_id").
		LeftJoin("nodes as nd ON quarantined_path_node_id = nd.node_id").
		Where("quarantined_path_registry_id = ? AND  i.image_name = ?", registryID, artifact)

	// Add version condition with proper parentheses for correct operator precedence
	stmtBuilder = stmtBuilder.Where(
		"(quarantined_path_artifact_id IS NULL OR "+
			"ar.artifact_version = ?)", version)

	// Add filepath condition with proper parentheses for correct operator precedence
	stmtBuilder = stmtBuilder.Where(
		"(quarantined_path_node_id IS NULL OR "+
			"nd.node_path = ?)",
		filePath)

	db := dbtx.GetAccessor(ctx, q.db)

	dst := []*QuarantineArtifactDB{}
	sqlQuery, args, err := stmtBuilder.ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to convert query to sql")
	}

	if err := db.SelectContext(ctx, &dst, sqlQuery, args...); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "Failed to find quarantine artifacts with the provided criteria")
	}

	return q.mapToQuarantineArtifactList(ctx, dst)
}

func (q QuarantineArtifactDao) Create(ctx context.Context, artifact *types.QuarantineArtifact) error {
	const sqlQuery = `
		INSERT INTO quarantined_paths ( 
			quarantined_path_id,
			quarantined_path_node_id,
			quarantined_path_reason,
			quarantined_path_registry_id,
			quarantined_path_artifact_id,
			quarantined_path_image_id,
			quarantined_path_created_at,
			quarantined_path_created_by
		) VALUES (
			:quarantined_path_id,
			:quarantined_path_node_id,
			:quarantined_path_reason,
			:quarantined_path_registry_id,
			:quarantined_path_artifact_id,
			:quarantined_path_image_id,
			:quarantined_path_created_at,
			:quarantined_path_created_by
		) 
		ON CONFLICT (quarantined_path_node_id, 
		quarantined_path_registry_id, quarantined_path_artifact_id, quarantined_path_image_id)
		DO NOTHING
		RETURNING quarantined_path_id`

	db := dbtx.GetAccessor(ctx, q.db)
	query, arg, err := db.BindNamed(sqlQuery, q.mapToInternalQuarantineArtifact(ctx, artifact))
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Failed to bind quarantine artifact object")
	}

	if err = db.QueryRowContext(ctx, query, arg...).Scan(&artifact.ID); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return databaseg.ProcessSQLErrorf(ctx, err, "Insert query failed")
	}

	return nil
}

func (q QuarantineArtifactDao) mapToQuarantineArtifact(_ context.Context,
	dst *QuarantineArtifactDB) *types.QuarantineArtifact {
	var artifactID, imageID int64
	var nodeID *string
	if dst.ArtifactID != nil {
		artifactID = *dst.ArtifactID
	}

	if dst.ImageID != nil {
		imageID = *dst.ImageID
	}

	if dst.NodeID != nil {
		nodeID = dst.NodeID
	}

	return &types.QuarantineArtifact{
		ID:         dst.ID,
		NodeID:     nodeID,
		Reason:     dst.Reason,
		RegistryID: dst.RegistryID,
		ArtifactID: artifactID,
		ImageID:    imageID,
		CreatedAt:  time.Unix(dst.CreatedAt, 0),
		CreatedBy:  dst.CreatedBy,
	}
}

func (q QuarantineArtifactDao) mapToQuarantineArtifactList(ctx context.Context,
	dsts []*QuarantineArtifactDB) ([]*types.QuarantineArtifact, error) {
	result := make([]*types.QuarantineArtifact, 0, len(dsts))
	for _, dst := range dsts {
		item := q.mapToQuarantineArtifact(ctx, dst)
		result = append(result, item)
	}
	return result, nil
}

func (q QuarantineArtifactDao) mapToInternalQuarantineArtifact(ctx context.Context,
	in *types.QuarantineArtifact) *QuarantineArtifactDB {
	session, _ := request.AuthSessionFrom(ctx)

	if in.ID == "" {
		in.ID = uuid.New().String()
	}

	if in.CreatedAt.IsZero() {
		in.CreatedAt = time.Now()
	}

	if in.CreatedBy == 0 && session != nil {
		in.CreatedBy = session.Principal.ID
	}

	var artifactIDPtr *int64
	if in.ArtifactID != 0 {
		artifactID := in.ArtifactID
		artifactIDPtr = &artifactID
	}

	var imageIDPtr *int64
	if in.ImageID != 0 {
		imageID := in.ImageID
		imageIDPtr = &imageID
	}

	return &QuarantineArtifactDB{
		ID:         in.ID,
		NodeID:     in.NodeID,
		Reason:     in.Reason,
		RegistryID: in.RegistryID,
		ArtifactID: artifactIDPtr,
		ImageID:    imageIDPtr,
		CreatedAt:  in.CreatedAt.Unix(),
		CreatedBy:  in.CreatedBy,
	}
}

func (q QuarantineArtifactDao) DeleteByRegistryIDArtifactAndFilePath(ctx context.Context,
	registryID int64, artifactID *int64, imageID int64, nodeID *string) error {
	// Build the delete query
	stmtBuilder := databaseg.Builder.
		Delete("quarantined_paths").
		Where("quarantined_path_registry_id = ?"+
			" AND quarantined_path_image_id = ?", registryID,
			imageID)

	if artifactID != nil {
		stmtBuilder = stmtBuilder.Where("quarantined_path_artifact_id = ?", *artifactID)
	}

	if nodeID != nil {
		stmtBuilder = stmtBuilder.Where("quarantined_path_node_id = ?", *nodeID)
	}

	// Execute the query
	db := dbtx.GetAccessor(ctx, q.db)
	sql, args, err := stmtBuilder.ToSql()
	if err != nil {
		return errors.Wrap(err, "Failed to convert delete query to SQL")
	}

	_, err = db.ExecContext(ctx, sql, args...)
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Failed to delete quarantine artifact with the provided criteria")
	}

	return nil
}
