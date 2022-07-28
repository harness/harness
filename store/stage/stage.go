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
	"context"

	"github.com/drone/drone/core"
	"github.com/drone/drone/store/shared/db"
)

// New returns a new StageStore.
func New(database *db.DB) core.StageStore {
	return &stageStore{database}
}

type stageStore struct {
	db *db.DB
}

func (s *stageStore) List(ctx context.Context, id int64) ([]*core.Stage, error) {
	var out []*core.Stage
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		params := map[string]interface{}{
			"stage_build_id": id,
		}
		stmt, args, err := binder.BindNamed(queryBuild, params)
		if err != nil {
			return err
		}
		rows, err := queryer.Query(stmt, args...)
		if err != nil {
			return err
		}
		out, err = scanRows(rows)
		return err
	})
	return out, err
}

func (s *stageStore) ListState(ctx context.Context, state string) ([]*core.Stage, error) {
	var out []*core.Stage
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		params := map[string]interface{}{
			"stage_status": state,
		}
		query := queryState
		// this is a workaround because mysql does not support
		// partial or filtered indexes for low-cardinality values.
		// For mysql we use a separate table to track pending and
		// running jobs to avoid full table scans.
		if (state == "pending" || state == "running") &&
			s.db.Driver() == db.Mysql {
			query = queryStateMysql
		}
		stmt, args, err := binder.BindNamed(query, params)
		if err != nil {
			return err
		}
		rows, err := queryer.Query(stmt, args...)
		if err != nil {
			return err
		}
		out, err = scanRows(rows)
		return err
	})
	return out, err
}

func (s *stageStore) ListSteps(ctx context.Context, id int64) ([]*core.Stage, error) {
	var out []*core.Stage
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		params := map[string]interface{}{
			"stage_build_id": id,
		}
		stmt, args, err := binder.BindNamed(queryNumberWithSteps, params)
		if err != nil {
			return err
		}
		rows, err := queryer.Query(stmt, args...)
		if err != nil {
			return err
		}
		out, err = scanRowsWithSteps(rows)
		return err
	})
	return out, err
}

func (s *stageStore) ListIncomplete(ctx context.Context) ([]*core.Stage, error) {
	var out []*core.Stage
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		stmt := queryUnfinished
		// this is a workaround because mysql does not support
		// partial or filtered indexes for low-cardinality values.
		// For mysql we use a separate table to track pending and
		// running jobs to avoid full table scans.
		if s.db.Driver() == db.Mysql {
			stmt = queryUnfinishedMysql
		}
		rows, err := queryer.Query(stmt)
		if err != nil {
			return err
		}
		out, err = scanRows(rows)
		return err
	})
	return out, err
}

func (s *stageStore) Find(ctx context.Context, id int64) (*core.Stage, error) {
	out := &core.Stage{ID: id}
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		params := toParams(out)
		query, args, err := binder.BindNamed(queryKey, params)
		if err != nil {
			return err
		}
		row := queryer.QueryRow(query, args...)
		return scanRow(row, out)
	})
	return out, err
}

func (s *stageStore) FindNumber(ctx context.Context, id int64, number int) (*core.Stage, error) {
	out := &core.Stage{BuildID: id, Number: number}
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		params := toParams(out)
		query, args, err := binder.BindNamed(queryNumber, params)
		if err != nil {
			return err
		}
		row := queryer.QueryRow(query, args...)
		return scanRow(row, out)
	})
	return out, err
}

func (s *stageStore) Create(ctx context.Context, stage *core.Stage) error {
	if s.db.Driver() == db.Postgres {
		return s.createPostgres(ctx, stage)
	}
	return s.create(ctx, stage)
}

func (s *stageStore) create(ctx context.Context, stage *core.Stage) error {
	stage.Version = 1
	return s.db.Lock(func(execer db.Execer, binder db.Binder) error {
		params := toParams(stage)
		stmt, args, err := binder.BindNamed(stmtInsert, params)
		if err != nil {
			return err
		}
		res, err := execer.Exec(stmt, args...)
		if err != nil {
			return err
		}
		stage.ID, err = res.LastInsertId()
		return err
	})
}

func (s *stageStore) createPostgres(ctx context.Context, stage *core.Stage) error {
	stage.Version = 1
	return s.db.Lock(func(execer db.Execer, binder db.Binder) error {
		params := toParams(stage)
		stmt, args, err := binder.BindNamed(stmtInsertPg, params)
		if err != nil {
			return err
		}
		return execer.QueryRow(stmt, args...).Scan(&stage.ID)
	})
}

func (s *stageStore) Update(ctx context.Context, stage *core.Stage) error {
	versionNew := stage.Version + 1
	versionOld := stage.Version

	err := s.db.Lock(func(execer db.Execer, binder db.Binder) error {
		params := toParams(stage)
		params["stage_version_old"] = versionOld
		params["stage_version_new"] = versionNew
		stmt, args, err := binder.BindNamed(stmtUpdate, params)
		if err != nil {
			return err
		}
		res, err := execer.Exec(stmt, args...)
		if err != nil {
			return err
		}
		effected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if effected == 0 {
			return db.ErrOptimisticLock
		}
		return nil
	})
	if err == nil {
		stage.Version = versionNew
	}
	return err
}

const queryBase = `
SELECT
 stage_id
,stage_repo_id
,stage_build_id
,stage_number
,stage_name
,stage_kind
,stage_type
,stage_status
,stage_error
,stage_errignore
,stage_exit_code
,stage_limit
,stage_limit_repo
,stage_os
,stage_arch
,stage_variant
,stage_kernel
,stage_machine
,stage_started
,stage_stopped
,stage_created
,stage_updated
,stage_version
,stage_on_success
,stage_on_failure
,stage_depends_on
,stage_labels
FROM stages
`

const queryKey = queryBase + `
WHERE stage_id = :stage_id
`

const queryState = queryBase + `
WHERE stage_status = :stage_status
ORDER BY stage_id ASC
`

const queryStateMysql = queryBase + `
WHERE stage_id IN (SELECT stage_id FROM stages_unfinished)
  AND stage_status = :stage_status
ORDER BY stage_id ASC
`

const queryUnfinished = queryBase + `
WHERE stage_status IN ('pending','running')
ORDER BY stage_id ASC
`

const queryUnfinishedMysql = queryBase + `
WHERE stage_id IN (SELECT stage_id FROM stages_unfinished)
  AND stage_status IN ('pending','running')
ORDER BY stage_id ASC
`

const queryBuild = queryBase + `
WHERE stage_build_id = :stage_build_id
ORDER BY stage_number ASC
`

const queryNumber = queryBase + `
WHERE stage_build_id = :stage_build_id
  AND stage_number = :stage_number
`

const queryNumberWithSteps = `
SELECT
 stage_id
,stage_repo_id
,stage_build_id
,stage_number
,stage_name
,stage_kind
,stage_type
,stage_status
,stage_error
,stage_errignore
,stage_exit_code
,stage_limit
,stage_limit_repo
,stage_os
,stage_arch
,stage_variant
,stage_kernel
,stage_machine
,stage_started
,stage_stopped
,stage_created
,stage_updated
,stage_version
,stage_on_success
,stage_on_failure
,stage_depends_on
,stage_labels
,step_id
,step_stage_id
,step_number
,step_name
,step_status
,step_error
,step_errignore
,step_exit_code
,step_started
,step_stopped
,step_version
,step_depends_on
,step_image
,step_detached
,step_schema
FROM stages
  LEFT JOIN steps
	ON stages.stage_id=steps.step_stage_id
  WHERE stages.stage_build_id = :stage_build_id
ORDER BY
 stage_id ASC
,step_id ASC
`

const stmtUpdate = `
UPDATE stages
SET
 stage_status = :stage_status
,stage_error = :stage_error
,stage_errignore = :stage_errignore
,stage_exit_code = :stage_exit_code
,stage_os = :stage_os
,stage_arch = :stage_arch
,stage_variant = :stage_variant
,stage_kernel = :stage_kernel
,stage_machine = :stage_machine
,stage_started = :stage_started
,stage_stopped = :stage_stopped
,stage_created = :stage_created
,stage_updated = :stage_updated
,stage_version = :stage_version_new
,stage_on_success = :stage_on_success
,stage_on_failure = :stage_on_failure
,stage_depends_on = :stage_depends_on
,stage_labels = :stage_labels
WHERE stage_id = :stage_id
  AND stage_version = :stage_version_old
`

const stmtInsert = `
INSERT INTO stages (
 stage_repo_id
,stage_build_id
,stage_number
,stage_name
,stage_kind
,stage_type
,stage_status
,stage_error
,stage_errignore
,stage_exit_code
,stage_limit
,stage_limit_repo
,stage_os
,stage_arch
,stage_variant
,stage_kernel
,stage_machine
,stage_started
,stage_stopped
,stage_created
,stage_updated
,stage_version
,stage_on_success
,stage_on_failure
,stage_depends_on
,stage_labels
) VALUES (
 :stage_repo_id
,:stage_build_id
,:stage_number
,:stage_name
,:stage_kind
,:stage_type
,:stage_status
,:stage_error
,:stage_errignore
,:stage_exit_code
,:stage_limit
,:stage_limit_repo
,:stage_os
,:stage_arch
,:stage_variant
,:stage_kernel
,:stage_machine
,:stage_started
,:stage_stopped
,:stage_created
,:stage_updated
,:stage_version
,:stage_on_success
,:stage_on_failure
,:stage_depends_on
,:stage_labels
)
`

const stmtInsertPg = stmtInsert + `
RETURNING stage_id
`
