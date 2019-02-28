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

package step

import (
	"database/sql"

	"github.com/drone/drone/core"
	"github.com/drone/drone/store/shared/db"
)

// helper function converts the Step structure to a set
// of named query parameters.
func toParams(from *core.Step) map[string]interface{} {
	return map[string]interface{}{
		"step_id":        from.ID,
		"step_stage_id":  from.StageID,
		"step_number":    from.Number,
		"step_name":      from.Name,
		"step_status":    from.Status,
		"step_error":     from.Error,
		"step_errignore": from.ErrIgnore,
		"step_exit_code": from.ExitCode,
		"step_started":   from.Started,
		"step_stopped":   from.Stopped,
		"step_version":   from.Version,
	}
}

// helper function scans the sql.Row and copies the column
// values to the destination object.
func scanRow(scanner db.Scanner, dest *core.Step) error {
	return scanner.Scan(
		&dest.ID,
		&dest.StageID,
		&dest.Number,
		&dest.Name,
		&dest.Status,
		&dest.Error,
		&dest.ErrIgnore,
		&dest.ExitCode,
		&dest.Started,
		&dest.Stopped,
		&dest.Version,
	)
}

// helper function scans the sql.Row and copies the column
// values to the destination object.
func scanRows(rows *sql.Rows) ([]*core.Step, error) {
	defer rows.Close()

	steps := []*core.Step{}
	for rows.Next() {
		step := new(core.Step)
		err := scanRow(rows, step)
		if err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}
	return steps, nil
}
