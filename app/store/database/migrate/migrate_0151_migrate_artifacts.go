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

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

//nolint:gocognit,stylecheck,revive,staticcheck // have naming match migration version
func MigrateAfter_0153_migrate_artifacts(ctx context.Context, dbtx *sql.Tx, driverName string) error {
	log := log.Ctx(ctx)

	log.Info().Msg("starting artifacts migration...")

	const idBatchSize = 8000
	const processBatchSize = 1000
	const maxIterations = 10000
	totalProcessed := 0
	batchCount := 0

	for {
		batchCount++
		if batchCount > maxIterations {
			return fmt.Errorf("artifacts migration exceeded %d iterations (%d processed) — aborting to prevent infinite loop",
				maxIterations, totalProcessed)
		}
		log.Info().Int("batch", batchCount).Msg("fetching orphan manifest IDs")

		// Step 1: Get orphan manifest IDs (lightweight query).
		manifestIDs, err := getOrphanManifestIDs(ctx, dbtx, driverName, idBatchSize)
		if err != nil {
			return fmt.Errorf("failed to get orphan manifest IDs: %w", err)
		}

		if len(manifestIDs) == 0 {
			break // No more manifests to process
		}

		log.Info().Int("batch", batchCount).Int("orphan_count", len(manifestIDs)).Msg("found orphan manifests")

		// Step 2: Process in smaller sub-batches.
		for start := 0; start < len(manifestIDs); start += processBatchSize {
			end := start + processBatchSize
			if end > len(manifestIDs) {
				end = len(manifestIDs)
			}
			subBatchIDs := manifestIDs[start:end]

			manifests, err := getManifestsByIDs(ctx, dbtx, subBatchIDs)
			if err != nil {
				return fmt.Errorf("failed to get manifests by IDs: %w", err)
			}

			_, err = processManifestsBatch(ctx, dbtx, manifests)
			if err != nil {
				return fmt.Errorf("failed to process manifests batch: %w", err)
			}

			totalProcessed += len(manifests)

			log.Info().
				Int("batch", batchCount).
				Int("sub_batch_count", len(manifests)).
				Int("total", totalProcessed).
				Msg("processed sub-batch")
		}
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

// getOrphanManifestIDs returns manifest IDs that have an image but no matching artifact.
// This is a lightweight query that only fetches IDs.
//
//nolint:gosec
func getOrphanManifestIDs(
	ctx context.Context,
	dbtx *sql.Tx,
	driverName string,
	limit int64,
) ([]int64, error) {
	var encodeFunction string
	if driverName == "sqlite3" {
		encodeFunction = "hex(m.manifest_digest)"
	} else {
		encodeFunction = "encode(m.manifest_digest, 'hex')"
	}

	query := fmt.Sprintf(`
        SELECT m.manifest_id
        FROM manifests m
        JOIN images i
             ON i.image_registry_id = m.manifest_registry_id
             AND i.image_name = m.manifest_image_name
        JOIN registries r
             ON r.registry_id = i.image_registry_id
        WHERE r.registry_package_type IN ('DOCKER', 'HELM')
        AND NOT EXISTS (
            SELECT 1 FROM artifacts a
            WHERE a.artifact_image_id = i.image_id
              AND a.artifact_version = %s)
        LIMIT $1`, encodeFunction)

	rows, err := dbtx.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query orphan manifest IDs: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan manifest ID: %w", err)
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating manifest IDs: %w", err)
	}

	return ids, nil
}

// getManifestsByIDs fetches full manifest rows for a given set of IDs.
func getManifestsByIDs(ctx context.Context, dbtx *sql.Tx, ids []int64) ([]manifest, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	q := databaseg.Builder.
		Select(
			"manifest_id",
			"manifest_image_name",
			"manifest_registry_id",
			"manifest_digest",
			"manifest_created_at",
			"manifest_updated_at",
			"manifest_created_by",
			"manifest_updated_by",
		).
		From("manifests").
		Where(sq.Eq{"manifest_id": ids}).
		OrderBy("manifest_id")

	query, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build manifests by IDs query: %w", err)
	}

	rows, err := dbtx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query manifests by IDs: %w", err)
	}
	defer rows.Close()

	var manifests []manifest
	for rows.Next() {
		var m manifest
		if err := rows.Scan(
			&m.ID, &m.ImageName, &m.RegistryID, &m.Digest,
			&m.CreatedAt, &m.UpdatedAt, &m.CreatedBy, &m.UpdatedBy,
		); err != nil {
			return nil, fmt.Errorf("failed to scan manifest: %w", err)
		}
		manifests = append(manifests, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating manifests: %w", err)
	}

	return manifests, nil
}

func processManifestsBatch(ctx context.Context, dbtx *sql.Tx, manifests []manifest) (int64, error) {
	db := dbtx
	var lastManifestID int64

	for _, m := range manifests {
		var imageID int64
		q := databaseg.Builder.Select("image_id").
			From("images").
			Where("image_name = ? AND image_registry_id = ?", m.ImageName, m.RegistryID)

		query, args, err := q.ToSql()
		if err != nil {
			return lastManifestID, fmt.Errorf("failed to build image select query: %w", err)
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
					return lastManifestID, fmt.Errorf("failed to build image insert query: %w", err2)
				}

				err = db.QueryRowContext(ctx, query2, args2...).Scan(&imageID)
				if err != nil {
					return lastManifestID, fmt.Errorf("failed to insert image: %w", err)
				}
			} else {
				return lastManifestID, fmt.Errorf("failed to check for existing image: %w", err)
			}
		}

		digestHex := hex.EncodeToString(m.Digest)
		var artifactID int64

		q3 := databaseg.Builder.Select("artifact_id").
			From("artifacts").
			Where("artifact_image_id = ? AND artifact_version = ?", imageID, digestHex)

		query3, args3, err3 := q3.ToSql()
		if err3 != nil {
			return lastManifestID, fmt.Errorf("failed to build artifact select query: %w", err3)
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
				return lastManifestID, fmt.Errorf("failed to build artifact insert query: %w", err4)
			}

			_, err = db.ExecContext(ctx, query4, args4...)
			if err != nil {
				return lastManifestID, fmt.Errorf("failed to insert artifact: %w", err)
			}
		} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return lastManifestID, fmt.Errorf("failed to check for existing artifact: %w", err)
		}

		if lastManifestID < m.ID {
			lastManifestID = m.ID
		}
	}
	return lastManifestID, nil
}
