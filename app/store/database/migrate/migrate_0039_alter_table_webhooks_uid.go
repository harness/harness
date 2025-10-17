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
	"strings"

	"github.com/harness/gitness/store/database"
	"github.com/harness/gitness/types/check"

	"github.com/guregu/null"
	gonanoid "github.com/matoous/go-nanoid"
	"github.com/rs/zerolog/log"
)

//nolint:gocognit,stylecheck,revive,staticcheck // have naming match migration version
func migrateAfter_0039_alter_table_webhooks_uid(ctx context.Context, dbtx *sql.Tx) error {
	log := log.Ctx(ctx)

	log.Info().Msg("backfill webhook_uid column")

	// Unfortunately we have to process page by page in memory and can't process as we read:
	// - lib/pq doesn't support updating webhook while select statement is ongoing
	//   https://github.com/lib/pq/issues/635
	// - sqlite3 doesn't support DECLARE CURSOR functionality
	type webhook struct {
		id          int64
		spaceID     null.Int
		repoID      null.Int
		identifier  null.String
		displayName string
	}

	const pageSize = 1000
	buffer := make([]webhook, pageSize)
	page := 0
	nextPage := func() (int, error) {
		const selectQuery = `
			SELECT webhook_id, webhook_display_name, webhook_repo_id, webhook_space_id, webhook_uid
			FROM webhooks
			ORDER BY webhook_repo_id, webhook_space_id, webhook_id
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
			err = rows.Scan(&buffer[c].id, &buffer[c].displayName, &buffer[c].repoID, &buffer[c].spaceID, &buffer[c].identifier)
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

	// keep track of unique identifiers for a given parent in memory (ASSUMPTION: limited number of webhooks per repo)
	parentID := ""
	parentChildIdentifiers := map[string]bool{}

	for {
		n, err := nextPage()
		if err != nil {
			return fmt.Errorf("failed to read next batch of webhooks: %w", err)
		}

		if n == 0 {
			break
		}

		for i := 0; i < n; i++ {
			wh := buffer[i]

			// concatenate repoID + spaceID to get unique parent id (only used to identify same parents)
			newParentID := fmt.Sprintf("%d_%d", wh.repoID.ValueOrZero(), wh.spaceID.ValueOrZero())
			if newParentID != parentID {
				// new parent? reset child identifiers
				parentChildIdentifiers = map[string]bool{}
				parentID = newParentID
			}

			// in case of down migration we already have identifiers for webhooks
			if len(wh.identifier.ValueOrZero()) > 0 {
				parentChildIdentifiers[strings.ToLower(wh.identifier.String)] = true

				log.Info().Msgf(
					"skip migration of webhook %d with displayname %q as it has a non-empty identifier %q",
					wh.id,
					wh.displayName,
					wh.identifier.String,
				)
				continue
			}

			// try to generate unique id (adds random suffix if deterministic identifier derived from display name isn't unique)
			for try := 0; try < 5; try++ {
				randomize := try > 0
				newIdentifier, err := WebhookDisplayNameToIdentifier(wh.displayName, randomize)
				if err != nil {
					return fmt.Errorf("failed to migrate displayname: %w", err)
				}
				newIdentifierLower := strings.ToLower(newIdentifier)
				if !parentChildIdentifiers[newIdentifierLower] {
					parentChildIdentifiers[newIdentifierLower] = true
					wh.identifier = null.StringFrom(newIdentifier)
					break
				}
			}

			if len(wh.identifier.ValueOrZero()) == 0 {
				return fmt.Errorf("failed to find a unique identifier for webhook %d with displayname %q", wh.id, wh.displayName)
			}

			log.Info().Msgf(
				"[%s] migrate webhook %d with displayname %q to identifier %q",
				parentID,
				wh.id,
				wh.displayName,
				wh.identifier.String,
			)

			const updateQuery = `
				UPDATE webhooks
				SET
					webhook_uid = $1
				WHERE
					webhook_id = $2`

			result, err := dbtx.ExecContext(ctx, updateQuery, wh.identifier.String, wh.id)
			if err != nil {
				return database.ProcessSQLErrorf(ctx, err, "failed to update webhook")
			}

			count, err := result.RowsAffected()
			if err != nil {
				return database.ProcessSQLErrorf(ctx, err, "Failed to get number of updated rows")
			}

			if count == 0 {
				return fmt.Errorf("failed to update webhook identifier - no rows were updated")
			}
		}
	}

	return nil
}

// WebhookDisplayNameToIdentifier migrates the provided displayname to a webhook identifier.
// If randomize is true, a random suffix is added to randomize the identifier.
//
//nolint:gocognit
func WebhookDisplayNameToIdentifier(displayName string, randomize bool) (string, error) {
	const placeholder = '_'
	const specialChars = ".-_"
	// remove / replace any illegal characters
	// Identifier Regex: ^[a-zA-Z0-9-_.]*$
	identifier := strings.Map(func(r rune) rune {
		switch {
		// drop any control characters or empty characters
		case r < 32 || r == 127:
			return -1

		// keep all allowed character
		case ('a' <= r && r <= 'z') ||
			('A' <= r && r <= 'Z') ||
			('0' <= r && r <= '9') ||
			strings.ContainsRune(specialChars, r):
			return r

		// everything else is replaced with the placeholder
		default:
			return placeholder
		}
	}, displayName)

	// remove any leading/trailing special characters
	identifier = strings.Trim(identifier, specialChars)

	// ensure string doesn't start with numbers (leading '_' is valid)
	if len(identifier) > 0 && identifier[0] >= '0' && identifier[0] <= '9' {
		identifier = string(placeholder) + identifier
	}

	// remove consecutive special characters
	identifier = santizeConsecutiveChars(identifier, specialChars)

	// ensure length restrictions
	if len(identifier) > check.MaxIdentifierLength {
		identifier = identifier[0:check.MaxIdentifierLength]
	}

	// backfill randomized identifier if sanitization ends up with empty identifier
	if len(identifier) == 0 {
		identifier = "webhook"
		randomize = true
	}

	if randomize {
		return randomizeIdentifier(identifier)
	}

	return identifier, nil
}

func santizeConsecutiveChars(in string, charSet string) string {
	if len(in) == 0 {
		return ""
	}

	inSet := func(b byte) bool {
		return strings.ContainsRune(charSet, rune(b))
	}

	out := strings.Builder{}
	out.WriteByte(in[0])
	for i := 1; i < len(in); i++ {
		if inSet(in[i]) && inSet(in[i-1]) {
			continue
		}
		out.WriteByte(in[i])
	}

	return out.String()
}

func randomizeIdentifier(identifier string) (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	const length = 4
	const maxLength = check.MaxIdentifierLength - length - 1 // max length of identifier to fit random suffix

	if len(identifier) > maxLength {
		identifier = identifier[0:maxLength]
	}
	suffix, err := gonanoid.Generate(alphabet, length)
	if err != nil {
		return "", fmt.Errorf("failed to generate gonanoid: %w", err)
	}

	return identifier + "_" + suffix, nil
}
