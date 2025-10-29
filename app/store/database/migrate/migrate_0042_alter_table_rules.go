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
	"encoding/json"
	"fmt"

	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/store/database"

	"github.com/rs/zerolog/log"
)

//nolint:gocognit,stylecheck,revive,staticcheck // have naming match migration version
func migrateAfter_0042_alter_table_rules(
	ctx context.Context,
	dbtx *sql.Tx,
) error {
	log := log.Ctx(ctx)

	log.Info().Msg("migrate all branch rules from uid to identifier")

	// Unfortunately we have to process page by page in memory and can't process as we read:
	// - lib/pq doesn't support updating rule while select statement is ongoing
	//   https://github.com/lib/pq/issues/635
	// - sqlite3 doesn't support DECLARE CURSOR functionality
	type rule struct {
		id         int64
		uid        string
		definition string
	}

	const pageSize = 1000
	buffer := make([]rule, pageSize)
	page := 0
	nextPage := func() (int, error) {
		const selectQuery = `
			SELECT rule_id, rule_uid, rule_definition
			FROM rules
			WHERE rule_type = 'branch'
			LIMIT $1
			OFFSET $2
			`
		rows, err := dbtx.QueryContext(ctx, selectQuery, pageSize, page*pageSize)
		if rows != nil {
			defer func() {
				err := rows.Close()
				if err != nil {
					log.Warn().Err(err).Msg("failed to close result rows")
				}
			}()
		}
		if err != nil {
			return 0, database.ProcessSQLErrorf(ctx, err, "failed batch select query")
		}

		c := 0
		for rows.Next() {
			err = rows.Scan(&buffer[c].id, &buffer[c].uid, &buffer[c].definition)
			if err != nil {
				return 0, database.ProcessSQLErrorf(ctx, err, "failed scanning next row")
			}
			c++
		}

		if rows.Err() != nil {
			return 0, database.ProcessSQLErrorf(ctx, err, "failed reading all rows")
		}

		page++

		return c, nil
	}

	for {
		n, err := nextPage()
		if err != nil {
			return fmt.Errorf("failed to read next batch of rules: %w", err)
		}

		if n == 0 {
			break
		}

		for i := range n {
			r := buffer[i]

			log.Info().Msgf(
				"migrate rule %d with identifier %q",
				r.id,
				r.uid,
			)

			branchDefinition := protection.Branch{}

			// unmarshaling the json will deserialize require_uids into require_uids and require_identifiers.
			// NOTE: could be done with existing SanitizeJSON method, but that would require dependencies.
			err = json.Unmarshal(json.RawMessage(r.definition), &branchDefinition)
			if err != nil {
				return fmt.Errorf("failed to unmarshal branch definition: %w", err)
			}

			updatedDefinitionRaw, err := protection.ToJSON(&branchDefinition)
			if err != nil {
				return fmt.Errorf("failed to marshal branch definition: %w", err)
			}

			// skip updating DB in case there's no change (e.g. no required checks are configured or migration re-runs)
			updatedDefinitionString := string(updatedDefinitionRaw)
			if updatedDefinitionString == r.definition {
				log.Info().Msg("skip updating rule as there's no change in definition")
				continue
			}

			const updateQuery = `
				UPDATE rules
				SET
					rule_definition = $1
				WHERE
					rule_id = $2`

			result, err := dbtx.ExecContext(ctx, updateQuery, updatedDefinitionString, r.id)
			if err != nil {
				return database.ProcessSQLErrorf(ctx, err, "failed to update rule")
			}

			count, err := result.RowsAffected()
			if err != nil {
				return database.ProcessSQLErrorf(ctx, err, "failed to get number of updated rows")
			}

			if count == 0 {
				return fmt.Errorf("failed to update branch rule definition - no rows were updated")
			}
		}
	}

	return nil
}
