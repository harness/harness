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
	"encoding/json"

	databaseg "github.com/harness/gitness/store/database"
	"github.com/harness/gitness/store/database/dbtx"
)

// FindPackageIdentifiersByRepositoryURL finds package identifiers (image names) by repository URL.
// Uses normalizedRepositoryURLs field for fast indexed lookups.
func (a ArtifactDao) FindPackageIdentifiersByRepositoryURL(
	ctx context.Context, registryID int64, repositoryURL string,
) ([]string, error) {
	query := `
		SELECT DISTINCT i.image_name
		FROM artifacts a
		JOIN images i ON a.artifact_image_id = i.image_id
		WHERE i.image_registry_id = $1
		  AND a.artifact_deleted_at IS NULL
		  AND a.artifact_metadata->'normalizedRepositoryURLs' @> $2::jsonb
		ORDER BY i.image_name
	`

	// JSONB containment requires array format - use json.Marshal for proper escaping
	jsonBytes, err := json.Marshal([]string{repositoryURL})
	if err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "failed to marshal repository URL")
	}
	jsonbValue := string(jsonBytes)

	db := dbtx.GetAccessor(ctx, a.db)

	var identifiers []string
	err = db.SelectContext(ctx, &identifiers, query, registryID, jsonbValue)
	if err != nil {
		return nil, databaseg.ProcessSQLErrorf(ctx, err, "failed to find package identifiers by repository URL")
	}

	return identifiers, nil
}
