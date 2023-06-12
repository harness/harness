// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

//go:build pq
// +build pq

package database

import (
	"github.com/lib/pq"
	"github.com/pkg/errors"
)

func isSQLUniqueConstraintError(original error) bool {
	var pqErr *pq.Error
	if errors.As(original, &pqErr) {
		return pqErr.Code == "23505" // unique_violation
	}

	return false
}
