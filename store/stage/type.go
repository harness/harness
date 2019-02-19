// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package stage

import (
	"database/sql"

	"github.com/drone/drone/core"
)

type nullStep struct {
	ID        sql.NullInt64
	StageID   sql.NullInt64
	Number    sql.NullInt64
	Name      sql.NullString
	Status    sql.NullString
	Error     sql.NullString
	ErrIgnore sql.NullBool
	ExitCode  sql.NullInt64
	Started   sql.NullInt64
	Stopped   sql.NullInt64
	Version   sql.NullInt64
}

func (s *nullStep) value() *core.Step {
	return &core.Step{
		ID:        s.ID.Int64,
		StageID:   s.StageID.Int64,
		Number:    int(s.Number.Int64),
		Name:      s.Name.String,
		Status:    s.Status.String,
		Error:     s.Error.String,
		ErrIgnore: s.ErrIgnore.Bool,
		ExitCode:  int(s.ExitCode.Int64),
		Started:   s.Started.Int64,
		Stopped:   s.Stopped.Int64,
		Version:   s.Version.Int64,
	}
}
