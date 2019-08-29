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

package build

import (
	"context"

	"github.com/drone/drone/core"
	"github.com/drone/drone/store/shared/db"
)

// New returns a new Buildcore.
func New(db *db.DB) core.BuildStore {
	return &buildStore{db}
}

type buildStore struct {
	db *db.DB
}

// Find returns a build from the datacore.
func (s *buildStore) Find(ctx context.Context, id int64) (*core.Build, error) {
	out := &core.Build{ID: id}
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

// FindNumber returns a build from the datastore by build number.
func (s *buildStore) FindNumber(ctx context.Context, repo, number int64) (*core.Build, error) {
	out := &core.Build{Number: number, RepoID: repo}
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

// FindLast returns the last build from the datastore by ref.
func (s *buildStore) FindRef(ctx context.Context, repo int64, ref string) (*core.Build, error) {
	out := &core.Build{RepoID: repo, Ref: ref}
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		params := toParams(out)
		query, args, err := binder.BindNamed(queryRowRef, params)
		if err != nil {
			return err
		}
		row := queryer.QueryRow(query, args...)
		return scanRow(row, out)
	})
	return out, err
}

// List returns a list of builds from the datastore by repository id.
func (s *buildStore) List(ctx context.Context, repo int64, limit, offset int) ([]*core.Build, error) {
	var out []*core.Build
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		params := map[string]interface{}{
			"build_repo_id": repo,
			"limit":         limit,
			"offset":        offset,
		}
		stmt, args, err := binder.BindNamed(queryRepo, params)
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

// ListRef returns a list of builds from the datastore by ref.
func (s *buildStore) ListRef(ctx context.Context, repo int64, ref string, limit, offset int) ([]*core.Build, error) {
	var out []*core.Build
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		params := map[string]interface{}{
			"build_repo_id": repo,
			"build_ref":     ref,
			"limit":         limit,
			"offset":        offset,
		}
		stmt, args, err := binder.BindNamed(queryRef, params)
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

// Pending returns a list of pending builds from the datastore by repository id.
func (s *buildStore) Pending(ctx context.Context) ([]*core.Build, error) {
	var out []*core.Build
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		rows, err := queryer.Query(queryPending)
		if err != nil {
			return err
		}
		out, err = scanRows(rows)
		return err
	})
	return out, err
}

// Running returns a list of running builds from the datastore by repository id.
func (s *buildStore) Running(ctx context.Context) ([]*core.Build, error) {
	var out []*core.Build
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		rows, err := queryer.Query(queryRunning)
		if err != nil {
			return err
		}
		out, err = scanRows(rows)
		return err
	})
	return out, err
}

// Create persists a build to the datacore.
func (s *buildStore) Create(ctx context.Context, build *core.Build, stages []*core.Stage) error {
	if s.db.Driver() == db.Postgres {
		return s.createPostgres(ctx, build, stages)
	}
	return s.create(ctx, build, stages)
}

func (s *buildStore) create(ctx context.Context, build *core.Build, stages []*core.Stage) error {
	build.Version = 1
	return s.db.Update(func(execer db.Execer, binder db.Binder) error {
		params := toParams(build)
		stmt, args, err := binder.BindNamed(stmtInsert, params)
		if err != nil {
			return err
		}
		res, err := execer.Exec(stmt, args...)
		if err != nil {
			return err
		}
		build.ID, err = res.LastInsertId()
		if err != nil {
			return err
		}

		for _, stage := range stages {
			stage.Version = 1
			stage.BuildID = build.ID
			params := toStageParams(stage)
			stmt, args, err := binder.BindNamed(stmtStageInsert, params)
			if err != nil {
				return err
			}
			res, err := execer.Exec(stmt, args...)
			if err != nil {
				return err
			}
			stage.ID, err = res.LastInsertId()
		}
		return err
	})
}

func (s *buildStore) createPostgres(ctx context.Context, build *core.Build, stages []*core.Stage) error {
	build.Version = 1
	return s.db.Update(func(execer db.Execer, binder db.Binder) error {
		params := toParams(build)
		stmt, args, err := binder.BindNamed(stmtInsertPg, params)
		if err != nil {
			return err
		}
		err = execer.QueryRow(stmt, args...).Scan(&build.ID)
		if err != nil {
			return err
		}

		for _, stage := range stages {
			stage.Version = 1
			stage.BuildID = build.ID
			params := toStageParams(stage)
			stmt, args, err := binder.BindNamed(stmtStageInsertPg, params)
			if err != nil {
				return err
			}
			err = execer.QueryRow(stmt, args...).Scan(&stage.ID)
			if err != nil {
				return err
			}
		}
		return err
	})
}

// Update updates a build in the datacore.
func (s *buildStore) Update(ctx context.Context, build *core.Build) error {
	versionNew := build.Version + 1
	versionOld := build.Version

	err := s.db.Lock(func(execer db.Execer, binder db.Binder) error {
		params := toParams(build)
		params["build_version_old"] = versionOld
		params["build_version_new"] = versionNew
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
		build.Version = versionNew
	}
	return err
}

// Delete deletes a build from the datacore.
func (s *buildStore) Delete(ctx context.Context, build *core.Build) error {
	return s.db.Lock(func(execer db.Execer, binder db.Binder) error {
		params := toParams(build)
		stmt, args, err := binder.BindNamed(stmtDelete, params)
		if err != nil {
			return err
		}
		_, err = execer.Exec(stmt, args...)
		return err
	})
}

// Purge deletes builds from the database where the build number is less than n.
func (s *buildStore) Purge(ctx context.Context, repo, number int64) error {
	build := &core.Build{
		RepoID: repo,
		Number: number,
	}
	return s.db.Lock(func(execer db.Execer, binder db.Binder) error {
		params := toParams(build)
		stmt, args, err := binder.BindNamed(stmtPurge, params)
		if err != nil {
			return err
		}
		_, err = execer.Exec(stmt, args...)
		return err
	})
}

// Count returns a count of builds.
func (s *buildStore) Count(ctx context.Context) (i int64, err error) {
	err = s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		return queryer.QueryRow(queryCount).Scan(&i)
	})
	return
}

const queryCount = `
SELECT COUNT(*)
FROM builds
`

const queryBase = `
SELECT
 build_id
,build_repo_id
,build_trigger
,build_number
,build_parent
,build_status
,build_error
,build_event
,build_action
,build_link
,build_timestamp
,build_title
,build_message
,build_before
,build_after
,build_ref
,build_source_repo
,build_source
,build_target
,build_author
,build_author_name
,build_author_email
,build_author_avatar
,build_sender
,build_params
,build_cron
,build_deploy
,build_deploy_id
,build_started
,build_finished
,build_created
,build_updated
,build_version
`

const queryKey = queryBase + `
FROM builds
WHERE build_id = :build_id
`

const queryNumber = queryBase + `
FROM builds
WHERE build_repo_id = :build_repo_id
  AND build_number = :build_number
`

const queryRef = queryBase + `
FROM builds
WHERE build_repo_id = :build_repo_id
  AND build_ref = :build_ref
ORDER BY build_id DESC
LIMIT :limit OFFSET :offset
`

const queryRowRef = queryBase + `
FROM builds
WHERE build_repo_id = :build_repo_id
  AND build_ref = :build_ref
ORDER BY build_id DESC
LIMIT 1
`

const queryRepo = queryBase + `
FROM builds
WHERE build_repo_id = :build_repo_id
ORDER BY build_id DESC
LIMIT :limit OFFSET :offset
`

const queryPending = queryBase + `
FROM builds
WHERE EXISTS (
    SELECT stage_id
    FROM stages
    WHERE stages.stage_build_id = builds.build_id
    AND stages.stage_status = 'pending'
)
ORDER BY build_id ASC
`

const queryRunning = queryBase + `
FROM builds
WHERE EXISTS (
    SELECT stage_id
    FROM stages
    WHERE stages.stage_build_id = builds.build_id
    AND stages.stage_status = 'running'
)
ORDER BY build_id ASC
`

// const queryRunningOLD = queryBase + `
// FROM builds
// WHERE build_status = 'running'
// ORDER BY build_id ASC
// `

const queryAll = queryBase + `
FROM builds
WHERE build_id > :build_id
LIMIT :limit OFFSET :offset
`

const stmtUpdate = `
UPDATE builds SET
 build_parent = :build_parent
,build_status = :build_status
,build_error = :build_error
,build_event = :build_event
,build_action = :build_action
,build_link = :build_link
,build_timestamp = :build_timestamp 
,build_title = :build_title
,build_message = :build_message
,build_before = :build_before
,build_after = :build_after
,build_ref = :build_ref
,build_source_repo = :build_source_repo
,build_source = :build_source
,build_target = :build_target
,build_author = :build_author
,build_author_name = :build_author_name
,build_author_email = :build_author_email
,build_author_avatar = :build_author_avatar
,build_sender = :build_sender
,build_params = :build_params
,build_cron = :build_cron
,build_deploy = :build_deploy
,build_started = :build_started
,build_finished = :build_finished
,build_updated = :build_updated
,build_version = :build_version_new
WHERE build_id = :build_id
  AND build_version = :build_version_old
`

const stmtInsert = `
INSERT INTO builds (
 build_repo_id
,build_trigger
,build_number
,build_parent
,build_status
,build_error
,build_event
,build_action
,build_link
,build_timestamp
,build_title
,build_message
,build_before
,build_after
,build_ref
,build_source_repo
,build_source
,build_target
,build_author
,build_author_name
,build_author_email
,build_author_avatar
,build_sender
,build_params
,build_cron
,build_deploy
,build_deploy_id
,build_started
,build_finished
,build_created
,build_updated
,build_version
) VALUES (
 :build_repo_id
,:build_trigger
,:build_number
,:build_parent
,:build_status
,:build_error
,:build_event
,:build_action
,:build_link
,:build_timestamp
,:build_title
,:build_message
,:build_before
,:build_after
,:build_ref
,:build_source_repo
,:build_source
,:build_target
,:build_author
,:build_author_name
,:build_author_email
,:build_author_avatar
,:build_sender
,:build_params
,:build_cron
,:build_deploy
,:build_deploy_id
,:build_started
,:build_finished
,:build_created
,:build_updated
,:build_version
)
`

const stmtInsertPg = stmtInsert + `
RETURNING build_id
`

const stmtStageInsert = `
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

const stmtStageInsertPg = stmtStageInsert + `
RETURNING stage_id
`

const stmtDelete = `
DELETE FROM builds
WHERE build_id = :build_id
`

const stmtPurge = `
DELETE FROM builds
WHERE build_repo_id = :build_repo_id
AND build_number < :build_number
`
