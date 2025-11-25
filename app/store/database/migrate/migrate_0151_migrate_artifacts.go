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

package migrate

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"

	databaseg "github.com/harness/gitness/store/database"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

//nolint:gocognit,stylecheck,revive,staticcheck // have naming match migration version
func MigrateAfter_0153_migrate_artifacts(ctx context.Context, dbtx *sql.Tx) error {
	log := log.Ctx(ctx)

	mediaTypeIDs, err := getMediaTypeIDs(ctx, dbtx)
	if err != nil {
		return fmt.Errorf("failed to get media type IDs: %w", err)
	}

	if len(mediaTypeIDs) == 0 {
		log.Info().Msg("no relevant media types found, nothing to migrate")
		return nil
	}

	log.Info().Msg("starting artifacts migration...")

	const batchSize = 1000
	offset := 0
	totalProcessed := 0
	batchCount := 0

	for {
		batchCount++
		log.Info().Int("batch", batchCount).Int("offset", offset).Msg("processing batch")

		manifests, err := getManifestsBatch(ctx, dbtx, mediaTypeIDs, batchSize, offset)
		if err != nil {
			return fmt.Errorf("failed to get manifests batch: %w", err)
		}

		if len(manifests) == 0 {
			break // No more manifests to process
		}

		// Process the batch
		err = processManifestsBatch(ctx, dbtx, manifests)
		if err != nil {
			return fmt.Errorf("failed to process manifests batch: %w", err)
		}

		totalProcessed += len(manifests)
		offset += batchSize

		log.Info().
			Int("batch", batchCount).
			Int("count", len(manifests)).
			Int("total", totalProcessed).
			Msg("processed batch")
	}

	log.Info().Int("total_processed", totalProcessed).Msg("artifacts migration completed")
	return nil
}

type manifest struct {
	ID         int64
	ImageName  string
	RegistryID int64
	Digest     []byte
	CreatedAt  int64
	UpdatedAt  int64
	CreatedBy  int64
	UpdatedBy  int64
}

func getMediaTypeIDs(ctx context.Context, dbtx *sql.Tx) ([]int64, error) {
	query := `
        SELECT mt_id 
        FROM media_types 
        WHERE mt_media_type IN (
            'application/vnd.docker.distribution.manifest.list.v2+json',
            'application/vnd.oci.image.index.v1+json'
        )`

	rows, err := dbtx.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query media types: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan media type ID: %w", err)
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating media types: %w", err)
	}

	return ids, nil
}

func getManifestsBatch(
	ctx context.Context,
	dbtx *sql.Tx,
	mediaTypeIDs []int64,
	limit, offset int,
) ([]manifest, error) {
	if len(mediaTypeIDs) != 2 {
		return nil, fmt.Errorf("expected exactly 2 media type IDs, got %d", len(mediaTypeIDs))
	}

	query := `
        SELECT 
            m.manifest_id,
            m.manifest_image_name,
            m.manifest_registry_id,
            m.manifest_digest,
            m.manifest_created_at,
            m.manifest_updated_at,
            m.manifest_created_by,
            m.manifest_updated_by
        FROM manifests m
        WHERE m.manifest_media_type_id IN ($1, $2)
        ORDER BY m.manifest_id
        LIMIT $3 OFFSET $4`

	args := []any{
		mediaTypeIDs[0],
		mediaTypeIDs[1],
		limit,
		offset,
	}

	rows, err := dbtx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query manifests: %w", err)
	}
	defer rows.Close()

	var manifests []manifest
	for rows.Next() {
		var m manifest
		err := rows.Scan(
			&m.ID,
			&m.ImageName,
			&m.RegistryID,
			&m.Digest,
			&m.CreatedAt,
			&m.UpdatedAt,
			&m.CreatedBy,
			&m.UpdatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan manifest: %w", err)
		}
		manifests = append(manifests, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating manifests: %w", err)
	}

	return manifests, nil
}

func processManifestsBatch(ctx context.Context, dbtx *sql.Tx, manifests []manifest) error {
	db := dbtx

	for _, m := range manifests {
		var imageID int64
		q := databaseg.Builder.Select("image_id").
			From("images").
			Where("image_name = ? AND image_registry_id = ?", m.ImageName, m.RegistryID)

		query, args, err := q.ToSql()
		if err != nil {
			return fmt.Errorf("failed to build image select query: %w", err)
		}

		err = db.QueryRowContext(ctx, query, args...).Scan(&imageID)
		//nolint: nestif
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				imgUUID := uuid.NewString()

				q2 := databaseg.Builder.Insert("images").
					SetMap(map[string]interface{}{
						"image_name":        m.ImageName,
						"image_registry_id": m.RegistryID,
						"image_type":        nil,
						"image_labels":      nil,
						"image_enabled":     true,
						"image_created_at":  m.CreatedAt,
						"image_updated_at":  m.CreatedAt,
						"image_created_by":  m.CreatedBy,
						"image_updated_by":  m.UpdatedBy,
						"image_uuid":        imgUUID,
					}).
					Suffix("RETURNING image_id")

				query2, args2, err2 := q2.ToSql()
				if err2 != nil {
					return fmt.Errorf("failed to build image insert query: %w", err)
				}

				err = db.QueryRowContext(ctx, query2, args2...).Scan(&imageID)
				if err != nil {
					return fmt.Errorf("failed to insert image: %w", err)
				}
			} else {
				return fmt.Errorf("failed to check for existing image: %w", err)
			}

			digestHex := hex.EncodeToString(m.Digest)
			var artifactID int64

			q3 := databaseg.Builder.Select("artifact_id").
				From("artifacts").
				Where("artifact_image_id = ? AND artifact_version = ?", imageID, digestHex)

			query3, args3, err3 := q3.ToSql()
			if err3 != nil {
				return fmt.Errorf("failed to build artifact select query: %w", err)
			}

			artUUID := uuid.NewString()
			err = db.QueryRowContext(ctx, query3, args3...).Scan(&artifactID)
			if errors.Is(err, sql.ErrNoRows) {
				q4 := databaseg.Builder.Insert("artifacts").
					SetMap(map[string]interface{}{
						"artifact_version":    digestHex,
						"artifact_image_id":   imageID,
						"artifact_created_at": m.CreatedAt,
						"artifact_updated_at": m.CreatedAt,
						"artifact_created_by": m.CreatedBy,
						"artifact_updated_by": m.UpdatedBy,
						"artifact_metadata":   nil,
						"artifact_uuid":       artUUID,
					})

				query4, args4, err4 := q4.ToSql()
				if err4 != nil {
					return fmt.Errorf("failed to build artifact insert query: %w", err)
				}

				_, err = db.ExecContext(ctx, query4, args4...)
				if err != nil {
					return fmt.Errorf("failed to insert artifact: %w", err)
				}
			} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("failed to check for existing artifact: %w", err)
			}
		}
	}
	return nil
}
