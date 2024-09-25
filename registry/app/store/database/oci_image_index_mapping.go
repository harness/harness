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
	"errors"
	"fmt"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/app/store/database/util"
	"github.com/harness/gitness/registry/types"
	store2 "github.com/harness/gitness/store"
	databaseg "github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

type ociImageIndexMappingDao struct {
	db *sqlx.DB
}

func NewOCIImageIndexMappingDao(db *sqlx.DB) store.OCIImageIndexMappingRepository {
	return &ociImageIndexMappingDao{
		db: db,
	}
}

type ociImageIndexMappingDB struct {
	ID               int64  `db:"oci_mapping_id"`
	ParentManifestID int64  `db:"oci_mapping_parent_manifest_id"`
	ChildDigest      []byte `db:"oci_mapping_child_digest"`
	CreatedAt        int64  `db:"oci_mapping_created_at"`
	UpdatedAt        int64  `db:"oci_mapping_updated_at"`
	CreatedBy        int64  `db:"oci_mapping_created_by"`
	UpdatedBy        int64  `db:"oci_mapping_updated_by"`
}

func (dao *ociImageIndexMappingDao) Create(
	ctx context.Context,
	ociManifest *types.OCIImageIndexMapping,
) error {
	const sqlQuery = `
        INSERT INTO oci_image_index_mappings (
			oci_mapping_parent_manifest_id,
			oci_mapping_child_digest,
			oci_mapping_created_at,
			oci_mapping_updated_at,
			oci_mapping_created_by,
			oci_mapping_updated_by
        ) VALUES (
			:oci_mapping_parent_manifest_id,
			:oci_mapping_child_digest,
			:oci_mapping_created_at,
			:oci_mapping_updated_at,
			:oci_mapping_created_by,
			:oci_mapping_updated_by
        ) ON CONFLICT (oci_mapping_parent_manifest_id, oci_mapping_child_digest)
            DO NOTHING
            RETURNING oci_mapping_id`

	db := dbtx.GetAccessor(ctx, dao.db)
	internalManifest := mapToInternalOCIMapping(ctx, ociManifest)
	query, args, err := db.BindNamed(sqlQuery, internalManifest)
	if err != nil {
		return databaseg.ProcessSQLErrorf(ctx, err, "Bind query failed")
	}

	if err = db.QueryRowContext(ctx, query, args...).Scan(&ociManifest.ID); err != nil {
		err = databaseg.ProcessSQLErrorf(ctx, err, "QueryRowContext failed")
		if errors.Is(err, store2.ErrDuplicate) {
			return nil
		}
		return fmt.Errorf("inserting OCI image index mapping: %w", err)
	}
	return nil
}

func (dao *ociImageIndexMappingDao) GetAllByChildDigest(
	ctx context.Context, registryID int64, imageName string, childDigest types.Digest,
) ([]*types.OCIImageIndexMapping, error) {
	digestBytes, err := util.GetHexDecodedBytes(string(childDigest))
	if err != nil {
		return nil, fmt.Errorf("failed to get digest bytes: %w", err)
	}
	const sqlQuery = `
        SELECT
            oci_mapping_id,
            oci_mapping_parent_manifest_id,
            oci_mapping_child_digest,
            oci_mapping_created_at,
            oci_mapping_updated_at,
            oci_mapping_created_by,
            oci_mapping_updated_by
        FROM
            oci_image_index_mappings
            JOIN manifests ON manifests.manifest_id = oci_image_index_mappings.oci_mapping_parent_manifest_id
        WHERE
            manifest_registry_id = $1 AND
            manifest_image_name = $2 AND
            oci_mapping_child_digest = $3`

	db := dbtx.GetAccessor(ctx, dao.db)
	rows, err := db.QueryxContext(ctx, sqlQuery, registryID, imageName, digestBytes)
	if err != nil || rows.Err() != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "QueryxContext failed")
	}
	defer rows.Close()

	var manifests []*types.OCIImageIndexMapping
	for rows.Next() {
		var dbManifest ociImageIndexMappingDB
		if err := rows.StructScan(&dbManifest); err != nil {
			return nil, databaseg.ProcessSQLErrorf(ctx, err, "StructScan failed")
		}
		manifests = append(manifests, mapToExternalOCIManifest(&dbManifest))
	}
	return manifests, nil
}

func mapToInternalOCIMapping(ctx context.Context, in *types.OCIImageIndexMapping) *ociImageIndexMappingDB {
	if in.CreatedAt.IsZero() {
		in.CreatedAt = time.Now()
	}
	in.UpdatedAt = time.Now()
	session, _ := request.AuthSessionFrom(ctx)
	if in.CreatedBy == 0 {
		in.CreatedBy = session.Principal.ID
	}
	in.UpdatedBy = session.Principal.ID
	childBytes, err := types.GetDigestBytes(in.ChildManifestDigest)
	if err != nil {
		log.Error().Msgf("failed to get digest bytes: %v", err)
	}

	return &ociImageIndexMappingDB{
		ID:               in.ID,
		ParentManifestID: in.ParentManifestID,
		ChildDigest:      childBytes,
		CreatedAt:        in.CreatedAt.UnixMilli(),
		UpdatedAt:        in.UpdatedAt.UnixMilli(),
		CreatedBy:        in.CreatedBy,
		UpdatedBy:        in.UpdatedBy,
	}
}

func mapToExternalOCIManifest(in *ociImageIndexMappingDB) *types.OCIImageIndexMapping {
	childDgst := types.Digest(util.GetHexEncodedString(in.ChildDigest))
	parsedChildDigest, err := childDgst.Parse()
	if err != nil {
		log.Error().Msgf("failed to child parse digest: %v", err)
	}

	return &types.OCIImageIndexMapping{
		ID:                  in.ID,
		ParentManifestID:    in.ParentManifestID,
		ChildManifestDigest: parsedChildDigest,
		CreatedAt:           time.UnixMilli(in.CreatedAt),
		UpdatedAt:           time.UnixMilli(in.UpdatedAt),
		CreatedBy:           in.CreatedBy,
		UpdatedBy:           in.UpdatedBy,
	}
}
