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
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/harness/gitness/job"
	rpmmetadata "github.com/harness/gitness/registry/app/metadata/rpm"
	"github.com/harness/gitness/store/database"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

//nolint:gocognit,stylecheck,revive,staticcheck // have naming match migration version
func MigrateAfter_0155_migrate_rpm_artifacts(
	ctx context.Context, dbtx *sql.Tx, driverName string,
) error {
	log := log.Ctx(ctx)
	log.Info().Msg("starting artifacts migration...")

	const batchSize = 1000
	offset := 0
	totalProcessed := 0
	batchCount := 0

	for {
		batchCount++
		log.Info().Int("batch", batchCount).Int("offset", offset).Msg("processing batch")

		artifacts, err := getArtifactsBatch(ctx, dbtx, driverName, batchSize, offset)
		if err != nil {
			return fmt.Errorf("failed to get artifacts batch: %w", err)
		}

		if len(artifacts) == 0 {
			break // No more artifacts to process
		}

		// Process the batch
		err = processArtifactsBatch(ctx, dbtx, artifacts)
		if err != nil {
			return fmt.Errorf("failed to process artifacts batch: %w", err)
		}

		totalProcessed += len(artifacts)
		offset += batchSize

		log.Info().
			Int("batch", batchCount).
			Int("count", len(artifacts)).
			Int("total", totalProcessed).
			Msg("processed batch")
	}

	log.Info().Int("total_processed", totalProcessed).Msg("artifacts migration completed")
	return nil
}

func getArtifactsBatch(
	ctx context.Context,
	dbtx *sql.Tx,
	driverName string,
	limit, offset int,
) ([]artifact, error) {
	var query string

	if driverName == "sqliteDriverName" {
		query = `SELECT a.artifact_id, i.image_name, a.artifact_version, i.image_registry_id, a.artifact_metadata
        FROM artifacts a
        JOIN images i ON a.artifact_image_id = i.image_id
        WHERE COALESCE(json_extract(artifact_metadata, '$.file_metadata.epoch'), '') NOT IN ('0', '');        `
	} else {
		query = `SELECT a.artifact_id, i.image_name, a.artifact_version, i.image_registry_id, a.artifact_metadata
        FROM artifacts a
        JOIN images i ON a.artifact_image_id = i.image_id
        WHERE COALESCE(artifact_metadata->'file_metadata'->>'epoch', '') NOT IN ('0', '')
        LIMIT $1 OFFSET $2`
	}

	args := []any{
		limit,
		offset,
	}
	rows, err := dbtx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query artifacts: %w", err)
	}
	defer rows.Close()

	var artifacts []artifact
	for rows.Next() {
		var a artifact
		err := rows.Scan(
			&a.ID,
			&a.Name,
			&a.Version,
			&a.RegistryID,
			&a.Metadata,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan artifact: %w", err)
		}
		artifacts = append(artifacts, a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error fetching rows: %w", err)
	}

	return artifacts, nil
}

type artifact struct {
	ID         int64
	Name       string
	Version    string
	RegistryID int64
	Metadata   json.RawMessage
}

type node struct {
	ID           string
	Name         string
	RegistryID   int64
	IsFile       bool
	NodePath     string
	BlobID       *string
	ParentNodeID *string
	CreatedAt    int64
	CreatedBy    int64
}

func processArtifactsBatch(ctx context.Context, dbtx *sql.Tx, artifacts []artifact) error {
	db := dbtx

	for _, a := range artifacts {
		metadata := rpmmetadata.RpmMetadata{}
		err := json.Unmarshal(a.Metadata, &metadata)
		if err != nil {
			return err
		}
		// nolint:nestif
		if strings.HasPrefix(a.Version, metadata.FileMetadata.Epoch+"-") {
			_, version, ok := strings.Cut(a.Version, "-")
			if !ok {
				return fmt.Errorf(
					"failed to cut version: %s", a.Version)
			}
			filename := fmt.Sprintf("%s-%s.rpm", a.Name, version)
			metadata.Files[0].Filename = filename
			metadataJSON, err := json.Marshal(metadata)
			if err != nil {
				return fmt.Errorf("failed to marshal metadata: %w", err)
			}

			//-------- update metadata
			dbVersion := metadata.FileMetadata.Epoch + ":" + version
			updateMetadataQuery := `UPDATE artifacts
		    SET artifact_metadata = $1, artifact_version = $2
		    WHERE artifact_id = $3`

			_, err = db.ExecContext(ctx, updateMetadataQuery, metadataJSON, dbVersion, a.ID)
			if err != nil {
				return fmt.Errorf("failed to update artifact metadata: %w", err)
			}

			//-------- update nodes
			lastDotIndex := strings.LastIndex(a.Version, ".")
			versionNode := a.Version[:lastDotIndex]
			nodesQuery := `SELECT * FROM nodes WHERE node_registry_id = $1 AND node_path LIKE $2`
			rows, err := dbtx.QueryContext(ctx, nodesQuery, a.RegistryID, "/"+a.Name+"/"+versionNode+"%")
			if err != nil {
				return fmt.Errorf("failed to query nodes: %w", err)
			}
			defer rows.Close()

			var nodes []node
			for rows.Next() {
				var n node
				err := rows.Scan(
					&n.ID,
					&n.Name,
					&n.ParentNodeID,
					&n.RegistryID,
					&n.IsFile,
					&n.NodePath,
					&n.BlobID,
					&n.CreatedAt,
					&n.CreatedBy,
				)
				if err != nil {
					return fmt.Errorf("failed to scan artifact: %w", err)
				}
				nodes = append(nodes, n)
			}

			if err := rows.Err(); err != nil {
				return fmt.Errorf("error fetching rows: %w", err)
			}
			for _, n := range nodes {
				l := strings.LastIndex(version, ".")
				newNodeName := version[:l]
				if n.Name == versionNode {
					updateNodeNameQuery := `UPDATE nodes
				SET node_name = $1
				WHERE node_id = $2`
					_, err = db.ExecContext(ctx, updateNodeNameQuery, newNodeName, n.ID)
					if err != nil {
						return fmt.Errorf("failed to update node: %w", err)
					}
				}

				newPath := strings.Replace(n.NodePath, "/"+versionNode, "/"+newNodeName, 1)
				updateNodeQuery := `UPDATE nodes
				SET node_path = $1
				WHERE node_id = $2`
				_, err = db.ExecContext(ctx, updateNodeQuery, newPath, n.ID)
				if err != nil {
					return fmt.Errorf("failed to update node: %w", err)
				}

				if strings.HasSuffix(n.NodePath, ".rpm") {
					// -------- insert epoch node
					lastSlashIndex := strings.LastIndex(newPath, "/")
					p := newPath[:lastSlashIndex]
					epochNodePath := p + "/" + metadata.FileMetadata.Epoch
					epochNodeID := uuid.NewString()
					insertQuery := `INSERT INTO nodes 
                    (node_id, node_name, node_parent_id, node_registry_id, node_is_file, node_path, 
                     node_generic_blob_id, node_created_at, node_created_by)
                    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

					_, err = db.ExecContext(ctx, insertQuery,
						epochNodeID,
						metadata.FileMetadata.Epoch,
						n.ParentNodeID,
						n.RegistryID,
						false,
						epochNodePath,
						nil,
						n.CreatedAt,
						n.CreatedBy,
					)
					if err != nil {
						return fmt.Errorf("failed to insert node: %w", err)
					}

					// -----------  update file node
					newFileNodeName := a.Name + "-" + version + ".rpm"
					updateNodeNameQuery := `UPDATE nodes
				       SET node_name = $1, node_path = $2, node_parent_id = $3
				       WHERE node_id = $4`
					_, err = db.ExecContext(ctx, updateNodeNameQuery, newFileNodeName,
						epochNodePath+"/"+newFileNodeName, epochNodeID, n.ID)
					if err != nil {
						return fmt.Errorf("failed to update node: %w", err)
					}
				}
			}
		}
		err = scheduleIndexJob(ctx, dbtx, RegistrySyncInput{RegistryIDs: []int64{a.RegistryID}})
		if err != nil {
			return fmt.Errorf("failed to schedule rpm registry index job: %w", err)
		}
	}
	return nil
}

func scheduleIndexJob(ctx context.Context, dbtx *sql.Tx, input RegistrySyncInput) error {
	data, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal repository sync job input json: %w", err)
	}

	data = bytes.TrimSpace(data)
	var idRaw [10]byte
	if _, err := rand.Read(idRaw[:]); err != nil {
		return fmt.Errorf("could not generate rpm registry index job ID: %w", err)
	}

	id := base32.StdEncoding.EncodeToString(idRaw[:])
	jd := job.Definition{
		UID:        id,
		Type:       "rpm_registry_index",
		MaxRetries: 5,
		Timeout:    45 * time.Minute,
		Data:       string(data),
	}
	nowMilli := time.Now().UnixMilli()
	job := &job.Job{
		UID:                 jd.UID,
		Created:             nowMilli,
		Updated:             nowMilli,
		Type:                jd.Type,
		Priority:            job.JobPriorityNormal,
		Data:                jd.Data,
		Result:              "",
		MaxDurationSeconds:  int(jd.Timeout / time.Second),
		MaxRetries:          jd.MaxRetries,
		State:               job.JobStateScheduled,
		Scheduled:           nowMilli,
		TotalExecutions:     0,
		RunBy:               "",
		RunDeadline:         nowMilli,
		RunProgress:         job.ProgressMin,
		LastExecuted:        0, // never executed
		IsRecurring:         false,
		RecurringCron:       "",
		ConsecutiveFailures: 0,
		LastFailureError:    "",
	}

	const sqlQuery = `INSERT INTO jobs (
    job_uid,
    job_created,
    job_updated,
    job_type,
    job_priority,
    job_data,
    job_result,
    job_max_duration_seconds,
    job_max_retries,
    job_state,
    job_scheduled,
    job_total_executions,
    job_run_by,
    job_run_deadline,
    job_run_progress,
    job_last_executed,
    job_is_recurring,
    job_recurring_cron,
    job_consecutive_failures,
    job_last_failure_error,
    job_group_id
		)  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)`

	if _, err := dbtx.ExecContext(ctx, sqlQuery, job.UID,
		job.Created,
		job.Updated,
		job.Type,
		job.Priority,
		job.Data,
		job.Result,
		job.MaxDurationSeconds,
		job.MaxRetries,
		job.State,
		job.Scheduled,
		job.TotalExecutions,
		job.RunBy,
		job.RunDeadline,
		job.RunProgress,
		job.LastExecuted,
		job.IsRecurring,
		job.RecurringCron,
		job.ConsecutiveFailures,
		job.LastFailureError,
		"",
	); err != nil {
		return database.ProcessSQLErrorf(ctx, err, "Insert job query failed")
	}
	return nil
}

type RegistrySyncInput struct {
	RegistryIDs []int64 `json:"registry_ids"`
}
