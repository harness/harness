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

package migrations

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/rs/zerolog/log"
)

// Migrate orchestrates database migrations using golang-migrate.
func Migrate(m *migrate.Migrate) error {
	version, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		log.Info().Msg("failed to fetch schema version from db.")
		return err
	}
	log.Info().Msgf("current version %d", version)
	if dirty {
		prev := int(version) - 1
		log.Info().Msg(fmt.Sprintf("schema is dirty at version = %d. Forcing version to %d", int(version), prev))
		err = m.Force(prev)
		if err != nil {
			log.Error().Stack().Err(err).Msg(fmt.Sprintf("failed to force schema version to %d %s", prev, err))
			return err
		}
	}
	err = m.Up()
	if errors.Is(err, migrate.ErrNoChange) {
		log.Info().Msg("No change to schema. No migrations were run")
		return nil
	}

	if err != nil && strings.Contains(err.Error(), "no migration found") {
		// The library throws this error when a give migration file does not exist. Unfortunately, we do not have
		// an error constant to compare with
		log.Error().Stack().Err(err).Msg("skipping migration because migration file was not found")
		return nil
	}

	if err != nil {
		log.Error().Stack().Err(err).Msg("failed to run db migrations")
		return fmt.Errorf("error when migration up: %w", err)
	}
	log.Info().Msg("Migrations successfully completed")
	return nil
}
