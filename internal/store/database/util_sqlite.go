// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

//go:build !pq
// +build !pq

package database

import (
	"github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

func isSQLUniqueConstraintError(original error) bool {
	var sqliteErr sqlite3.Error
	if errors.As(original, &sqliteErr) {
		return errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintUnique)
	}

	return false
}
