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

//go:build !nosqlite
// +build !nosqlite

package database

import (
	"github.com/jackc/pgerrcode"
	"github.com/lib/pq"
	"github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

func isSQLUniqueConstraintError(original error) bool {
	var sqliteErr sqlite3.Error
	if errors.As(original, &sqliteErr) {
		return errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintUnique) ||
			errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintPrimaryKey)
	}

	var pqErr *pq.Error
	if errors.As(original, &pqErr) {
		return pqErr.Code == "23505" // unique_violation
	}

	return false
}

func isSQLForeignKeyViolationError(original error) bool {
	var sqliteErr sqlite3.Error
	if errors.As(original, &sqliteErr) {
		return errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintForeignKey)
	}

	var pqErr *pq.Error
	// this can happen if the child manifest is deleted by
	// the online GC while attempting to create the list
	if errors.As(original, &pqErr) && pqErr.Code == pgerrcode.ForeignKeyViolation {
		return true
	}

	return false
}
