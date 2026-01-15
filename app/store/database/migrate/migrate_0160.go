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
	"fmt"

	"github.com/rs/zerolog/log"
)

//nolint:gocognit,stylecheck,revive,staticcheck
func MigrateAfter_0160(
	ctx context.Context, dbtx *sql.Tx,
) error {
	log := log.Ctx(ctx)
	log.Info().Msg("starting artifacts migration...")

	query := `SELECT r.registry_id from registries as r where r.registry_package_type = 'RPM'`

	rows, err := dbtx.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query registries: %w", err)
	}
	defer rows.Close()

	var registryIDs []int64
	for rows.Next() {
		var rID int64
		err := rows.Scan(&rID)
		if err != nil {
			return fmt.Errorf("failed to scan registry_id: %w", err)
		}
		registryIDs = append(registryIDs, rID)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error fetching rows: %w", err)
	}

	if err := scheduleIndexJob(ctx, dbtx, RegistrySyncInput{RegistryIDs: registryIDs}); err != nil {
		return fmt.Errorf("failed to schedule index job: %w", err)
	}
	log.Info().Msg("rpm registries 0160 migration completed")
	return nil
}
