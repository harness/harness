// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stage

import (
	"database/sql"
	"encoding/json"

	"github.com/drone/drone/core"
	"github.com/jmoiron/sqlx/types"
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
	DependsOn types.JSONText
	Image     sql.NullString
	Detached  sql.NullBool
	Schema    sql.NullString
}

func (s *nullStep) value() *core.Step {
	var dependsOn []string
	json.Unmarshal(s.DependsOn, &dependsOn)

	step := &core.Step{
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
		DependsOn: dependsOn,
		Image:     s.Image.String,
		Detached:  s.Detached.Bool,
		Schema:    s.Schema.String,
	}

	return step
}
