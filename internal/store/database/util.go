// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"database/sql"

	"github.com/harness/gitness/types/errs"
	"github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
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

func wrapSqlErrorf(original error, format string, args ...interface{}) error {
	if original == sql.ErrNoRows {
		original = errs.WrapInResourceNotFound(original)
	} else if isSqlUniqueConstraintError(original) {
		original = errs.WrapInDuplicate(original)
	}

	return errors.Wrapf(original, format, args...)
}

func isSqlUniqueConstraintError(original error) bool {
	o3, ok := original.(sqlite3.Error)
	return ok && errors.Is(o3.ExtendedCode, sqlite3.ErrConstraintUnique)
}
