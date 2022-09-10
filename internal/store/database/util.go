// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"database/sql"
	"fmt"

	"github.com/harness/gitness/internal/store"
	"github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// default query range limit.
const defaultLimit = 100

// limit returns the page size to a sql limit.
func limit(size int) int {
	if size == 0 {
		size = defaultLimit
	}
	return size
}

// offset converts the page to a sql offset.
func offset(page, size int) int {
	if page == 0 {
		page = 1
	}
	if size == 0 {
		size = defaultLimit
	}
	page = page - 1
	return page * size
}

// Logs the error and message, returns either the original error or a store equivalent if possible.
func processSqlErrorf(err error, format string, args ...interface{}) error {
	// always log DB error (print formated message)
	log.Warn().Msgf("%s %s", fmt.Sprintf(format, args...), err)

	// If it's a known error, return converted error instead.
	if err == sql.ErrNoRows {
		return store.ErrResourceNotFound
	} else if isSqlUniqueConstraintError(err) {
		return store.ErrDuplicate
	}

	return err
}

func isSqlUniqueConstraintError(original error) bool {
	o3, ok := original.(sqlite3.Error)
	return ok && errors.Is(o3.ExtendedCode, sqlite3.ErrConstraintUnique)
}
