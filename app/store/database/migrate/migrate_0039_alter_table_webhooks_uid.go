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

//nolint:gocognit,stylecheck,revive // have naming match migration version
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
		uid         null.String
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
			return 0, database.ProcessSQLErrorf(err, "failed batch select query")
		}

		c := 0
		for rows.Next() {
			err = rows.Scan(&buffer[c].id, &buffer[c].displayName, &buffer[c].repoID, &buffer[c].spaceID, &buffer[c].uid)
			if err != nil {
				return 0, database.ProcessSQLErrorf(err, "failed scanning next row")
			}
			c++
		}

		if rows.Err() != nil {
			return 0, database.ProcessSQLErrorf(err, "failed reading all rows")
		}

		page++

		return c, nil
	}

	// keep track of unique UIDs for a given parent in memory (ASSUMPTION: limited number of webhooks per repo)
	parentID := ""
	parentChildUIDs := map[string]bool{}

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

			// concatinate repoID + spaceID to get unique parent id (only used to identify same parents)
			newParentID := fmt.Sprintf("%d_%d", wh.repoID.ValueOrZero(), wh.spaceID.ValueOrZero())
			if newParentID != parentID {
				// new parent? reset child UIDs
				parentChildUIDs = map[string]bool{}
				parentID = newParentID
			}

			// in case of down migration we already have uids for webhooks
			if len(wh.uid.ValueOrZero()) > 0 {
				parentChildUIDs[strings.ToLower(wh.uid.String)] = true

				log.Info().Msgf(
					"skip migration of webhook %d with displayname %q as it has a non-empty uid %q",
					wh.id,
					wh.displayName,
					wh.uid.String,
				)
				continue
			}

			// try to generate unique id (adds random suffix if deterministic uid derived from display name isn't unique)
			for try := 0; try < 5; try++ {
				randomize := try > 0
				newUID, err := WebhookDisplayNameToUID(wh.displayName, randomize)
				if err != nil {
					return fmt.Errorf("failed to migrate displayname: %w", err)
				}
				newUIDLower := strings.ToLower(newUID)
				if !parentChildUIDs[newUIDLower] {
					parentChildUIDs[newUIDLower] = true
					wh.uid = null.StringFrom(newUID)
					break
				}
			}

			if len(wh.uid.ValueOrZero()) == 0 {
				return fmt.Errorf("failed to find a unique uid for webhook %d with displayname %q", wh.id, wh.displayName)
			}

			log.Info().Msgf(
				"[%s] migrate webhook %d with displayname %q to uid %q",
				parentID,
				wh.id,
				wh.displayName,
				wh.uid.String,
			)

			const updateQuery = `
				UPDATE webhooks
				SET
					webhook_uid = $1
				WHERE
					webhook_id = $2`

			result, err := dbtx.ExecContext(ctx, updateQuery, wh.uid.String, wh.id)
			if err != nil {
				return database.ProcessSQLErrorf(err, "failed to update webhook")
			}

			count, err := result.RowsAffected()
			if err != nil {
				return database.ProcessSQLErrorf(err, "Failed to get number of updated rows")
			}

			if count == 0 {
				return fmt.Errorf("failed to update webhook uid - no rows were updated")
			}
		}
	}

	return nil
}

// WebhookDisplayNameToUID migrates the provided displayname to a webhook uid.
// If randomize is true, a random suffix is added to randomize the uid.
//
//nolint:gocognit
func WebhookDisplayNameToUID(displayName string, randomize bool) (string, error) {
	const placeholder = '_'
	const specialChars = ".-_"
	// remove / replace any illegal characters
	// UID Regex: ^[a-zA-Z_][a-zA-Z0-9-_.]*$
	uid := strings.Map(func(r rune) rune {
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
	uid = strings.Trim(uid, specialChars)

	// ensure string doesn't start with numbers (leading '_' is valid)
	if len(uid) > 0 && uid[0] >= '0' && uid[0] <= '9' {
		uid = string(placeholder) + uid
	}

	// remove consecutive special characters
	uid = santizeConsecutiveChars(uid, specialChars)

	// ensure length restrictions
	if len(uid) > check.MaxUIDLength {
		uid = uid[0:check.MaxUIDLength]
	}

	// backfill randomized uid if sanitization ends up with empty uid
	if len(uid) == 0 {
		uid = "webhook"
		randomize = true
	}

	if randomize {
		return randomizeUID(uid)
	}

	return uid, nil
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

func randomizeUID(uid string) (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	const length = 4
	const maxLength = check.MaxUIDLength - length - 1 // max length of uid to fit random suffix

	if len(uid) > maxLength {
		uid = uid[0:maxLength]
	}
	suffix, err := gonanoid.Generate(alphabet, length)
	if err != nil {
		return "", fmt.Errorf("failed to generate gonanoid: %w", err)
	}

	return uid + "_" + suffix, nil
}
