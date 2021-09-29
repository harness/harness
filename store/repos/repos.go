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

package repos

import (
	"context"

	"github.com/drone/drone/core"
	"github.com/drone/drone/store/shared/db"
)

// New returns a new RepositoryStore.
func New(db *db.DB) core.RepositoryStore {
	return &repoStore{db}
}

type repoStore struct {
	db *db.DB
}

func (s *repoStore) List(ctx context.Context, id int64) ([]*core.Repository, error) {
	var out []*core.Repository
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		params := map[string]interface{}{"user_id": id}
		query, args, err := binder.BindNamed(queryPerms, params)
		if err != nil {
			return err
		}
		rows, err := queryer.Query(query, args...)
		if err != nil {
			return err
		}
		out, err = scanRows(rows)
		return err
	})
	return out, err
}

func (s *repoStore) ListLatest(ctx context.Context, id int64) ([]*core.Repository, error) {
	var out []*core.Repository
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		params := map[string]interface{}{
			"user_id":     id,
			"repo_active": true,
		}
		stmt := queryRepoWithBuild
		if s.db.Driver() == db.Postgres {
			stmt = queryRepoWithBuildPostgres
		}
		query, args, err := binder.BindNamed(stmt, params)
		if err != nil {
			return err
		}
		rows, err := queryer.Query(query, args...)
		if err != nil {
			return err
		}
		out, err = scanRowsBuild(rows)
		return err
	})
	return out, err
}

func (s *repoStore) ListRecent(ctx context.Context, id int64) ([]*core.Repository, error) {
	var out []*core.Repository
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		params := map[string]interface{}{"user_id": id}
		query, args, err := binder.BindNamed(queryRepoWithBuildAll, params)
		if err != nil {
			return err
		}
		rows, err := queryer.Query(query, args...)
		if err != nil {
			return err
		}
		out, err = scanRowsBuild(rows)
		return err
	})
	return out, err
}

func (s *repoStore) ListIncomplete(ctx context.Context) ([]*core.Repository, error) {
	var out []*core.Repository
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		rows, err := queryer.Query(queryRepoWithBuildIncomplete)
		if err != nil {
			return err
		}
		out, err = scanRowsBuild(rows)
		return err
	})
	return out, err
}

func (s *repoStore) ListRunningStatus(ctx context.Context) ([]*core.RepoBuildStage, error) {
	var out []*core.RepoBuildStage
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		rows, err := queryer.Query(queryReposRunningStatus)
		if err != nil {
			return err
		}
		out, err = repoBuildStageRowsBuild(rows)
		return err
	})
	return out, err
}

func (s *repoStore) ListAll(ctx context.Context, limit, offset int) ([]*core.Repository, error) {
	var out []*core.Repository
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		params := map[string]interface{}{
			"limit":  limit,
			"offset": offset,
		}
		query, args, err := binder.BindNamed(queryAll, params)
		if err != nil {
			return err
		}
		rows, err := queryer.Query(query, args...)
		if err != nil {
			return err
		}
		out, err = scanRows(rows)
		return err
	})
	return out, err
}

func (s *repoStore) Find(ctx context.Context, id int64) (*core.Repository, error) {
	out := &core.Repository{ID: id}
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		params := ToParams(out)
		query, args, err := binder.BindNamed(queryKey, params)
		if err != nil {
			return err
		}
		row := queryer.QueryRow(query, args...)
		return scanRow(row, out)
	})
	return out, err
}

func (s *repoStore) FindName(ctx context.Context, namespace, name string) (*core.Repository, error) {
	out := &core.Repository{Slug: namespace + "/" + name}
	err := s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		params := ToParams(out)
		query, args, err := binder.BindNamed(querySlug, params)
		if err != nil {
			return err
		}
		row := queryer.QueryRow(query, args...)
		return scanRow(row, out)
	})
	return out, err
}

func (s *repoStore) Create(ctx context.Context, repo *core.Repository) error {
	if s.db.Driver() == db.Postgres {
		return s.createPostgres(ctx, repo)
	}
	return s.create(ctx, repo)
}

func (s *repoStore) create(ctx context.Context, repo *core.Repository) error {
	return s.db.Lock(func(execer db.Execer, binder db.Binder) error {
		repo.Version = 1 // set the initial record version
		params := ToParams(repo)
		stmt, args, err := binder.BindNamed(stmtInsert, params)
		if err != nil {
			return err
		}
		res, err := execer.Exec(stmt, args...)
		if err != nil {
			return err
		}
		repo.ID, err = res.LastInsertId()
		return err
	})
}

func (s *repoStore) createPostgres(ctx context.Context, repo *core.Repository) error {
	return s.db.Lock(func(execer db.Execer, binder db.Binder) error {
		repo.Version = 1 // set the initial record version
		params := ToParams(repo)
		stmt, args, err := binder.BindNamed(stmtInsertPg, params)
		if err != nil {
			return err
		}
		return execer.QueryRow(stmt, args...).Scan(&repo.ID)
	})
}

func (s *repoStore) Activate(ctx context.Context, repo *core.Repository) error {
	return s.Update(ctx, repo)
}

func (s *repoStore) Update(ctx context.Context, repo *core.Repository) error {
	versionNew := repo.Version + 1
	versionOld := repo.Version
	err := s.db.Lock(func(execer db.Execer, binder db.Binder) error {
		params := ToParams(repo)
		params["repo_version_old"] = versionOld
		params["repo_version_new"] = versionNew
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
		repo.Version = versionNew
	}
	return err
}

func (s *repoStore) Delete(ctx context.Context, repo *core.Repository) error {
	return s.db.Lock(func(execer db.Execer, binder db.Binder) error {
		params := ToParams(repo)
		stmt, args, _ := binder.BindNamed(stmtDelete, params)
		_, err := execer.Exec(stmt, args...)
		return err
	})
}

func (s *repoStore) Count(ctx context.Context) (i int64, err error) {
	err = s.db.View(func(queryer db.Queryer, binder db.Binder) error {
		params := map[string]interface{}{"repo_active": true}
		query, args, _ := binder.BindNamed(queryCount, params)
		return queryer.QueryRow(query, args...).Scan(&i)
	})
	return
}

func (s *repoStore) Increment(ctx context.Context, repo *core.Repository) (*core.Repository, error) {
	for {
		repo.Counter++
		err := s.Update(ctx, repo)
		if err == nil {
			return repo, nil
		}
		if err != nil && err != db.ErrOptimisticLock {
			return repo, err
		}
		repo, err = s.Find(ctx, repo.ID)
		if err != nil {
			return nil, err
		}
	}
}

const queryCount = `
SELECT count(*)
FROM repos
WHERE repo_active = :repo_active
`

const queryCols = `
SELECT
 repo_id
,repo_uid
,repo_user_id
,repo_namespace
,repo_name
,repo_slug
,repo_scm
,repo_clone_url
,repo_ssh_url
,repo_html_url
,repo_active
,repo_private
,repo_visibility
,repo_branch
,repo_counter
,repo_config
,repo_timeout
,repo_throttle
,repo_trusted
,repo_protected
,repo_no_forks
,repo_no_pulls
,repo_cancel_pulls
,repo_cancel_push
,repo_cancel_running
,repo_synced
,repo_created
,repo_updated
,repo_version
,repo_signer
,repo_secret
`

const queryColsBuilds = queryCols + `
,build_id
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
,build_debug
,build_started
,build_finished
,build_created
,build_updated
,build_version
`

const queryKey = queryCols + `
FROM repos
WHERE repo_id = :repo_id
`

const querySlug = queryCols + `
FROM repos
WHERE repo_slug = :repo_slug
`

const queryPerms = queryCols + `
FROM repos
INNER JOIN perms ON perms.perm_repo_uid = repos.repo_uid
WHERE perms.perm_user_id = :user_id
ORDER BY repo_slug ASC
`

const queryAll = queryCols + `
FROM repos
LIMIT :limit OFFSET :offset
`

const stmtDelete = `
DELETE FROM repos WHERE repo_id = :repo_id
`

const stmtInsert = `
INSERT INTO repos (
 repo_uid
,repo_user_id
,repo_namespace
,repo_name
,repo_slug
,repo_scm
,repo_clone_url
,repo_ssh_url
,repo_html_url
,repo_active
,repo_private
,repo_visibility
,repo_branch
,repo_counter
,repo_config
,repo_timeout
,repo_throttle
,repo_trusted
,repo_protected
,repo_no_forks
,repo_no_pulls
,repo_cancel_pulls
,repo_cancel_push
,repo_cancel_running
,repo_synced
,repo_created
,repo_updated
,repo_version
,repo_signer
,repo_secret
) VALUES (
 :repo_uid
,:repo_user_id
,:repo_namespace
,:repo_name
,:repo_slug
,:repo_scm
,:repo_clone_url
,:repo_ssh_url
,:repo_html_url
,:repo_active
,:repo_private
,:repo_visibility
,:repo_branch
,:repo_counter
,:repo_config
,:repo_timeout
,:repo_throttle
,:repo_trusted
,:repo_protected
,:repo_no_forks
,:repo_no_pulls
,:repo_cancel_pulls
,:repo_cancel_push
,:repo_cancel_running
,:repo_synced
,:repo_created
,:repo_updated
,:repo_version
,:repo_signer
,:repo_secret
)
`

const stmtInsertPg = stmtInsert + `
RETURNING repo_id
`

const stmtPermInsert = `
INSERT INTO perms VALUES (
 :perm_user_id
,:perm_repo_uid
,:perm_read
,:perm_write
,:perm_admin
,:perm_synced
,:perm_created
,:perm_updated
)
`

const stmtUpdate = `
UPDATE repos SET
 repo_user_id = :repo_user_id
,repo_namespace = :repo_namespace
,repo_name = :repo_name
,repo_slug = :repo_slug
,repo_scm = :repo_scm
,repo_clone_url = :repo_clone_url
,repo_ssh_url = :repo_ssh_url
,repo_html_url = :repo_html_url
,repo_branch = :repo_branch
,repo_private = :repo_private
,repo_visibility = :repo_visibility
,repo_active = :repo_active
,repo_config = :repo_config
,repo_trusted = :repo_trusted
,repo_protected = :repo_protected
,repo_no_forks = :repo_no_forks
,repo_no_pulls = :repo_no_pulls
,repo_cancel_pulls = :repo_cancel_pulls
,repo_cancel_push = :repo_cancel_push
,repo_cancel_running = :repo_cancel_running
,repo_timeout = :repo_timeout
,repo_throttle = :repo_throttle
,repo_counter = :repo_counter
,repo_synced = :repo_synced
,repo_created = :repo_created
,repo_updated = :repo_updated
,repo_version = :repo_version_new
,repo_signer = :repo_signer
,repo_secret = :repo_secret
WHERE repo_id = :repo_id
  AND repo_version = :repo_version_old
`

// TODO(bradrydzewski) this query needs performance tuning.
// one approach that is promising is the ability to use the
// repo_counter (latest build number) to join on the build
// table.
//
//   FROM repos LEFT OUTER JOIN builds ON (
//     repos.repo_id = builds.build_repo_id AND
//     builds.build_number = repos.repo_counter
//   )
//   INNER JOIN perms ON perms.perm_repo_uid = repos.repo_uid
//

const queryRepoWithBuild = queryColsBuilds + `
FROM repos LEFT OUTER JOIN builds ON build_id = (
	SELECT build_id FROM builds
	WHERE builds.build_repo_id = repos.repo_id
	ORDER BY build_id DESC
	LIMIT 1
)
INNER JOIN perms ON perms.perm_repo_uid = repos.repo_uid
WHERE perms.perm_user_id = :user_id
ORDER BY repo_slug ASC
`

const queryRepoWithBuildPostgres = queryColsBuilds + `
FROM repos LEFT OUTER JOIN builds ON build_id = (
	SELECT DISTINCT ON (build_repo_id) build_id FROM builds
	WHERE builds.build_repo_id = repos.repo_id
	ORDER BY build_repo_id, build_id DESC
)
INNER JOIN perms ON perms.perm_repo_uid = repos.repo_uid
WHERE perms.perm_user_id = :user_id
ORDER BY repo_slug ASC
`

const queryRepoWithBuildAll = queryColsBuilds + `
FROM repos
INNER JOIN perms  ON perms.perm_repo_uid = repos.repo_uid
INNER JOIN builds ON builds.build_repo_id = repos.repo_id
WHERE perms.perm_user_id = :user_id
ORDER BY build_id DESC
LIMIT 25;
`

const queryRepoWithBuildIncomplete = queryColsBuilds + `
FROM repos
INNER JOIN builds ON builds.build_repo_id = repos.repo_id
WHERE EXISTS (
    SELECT stage_id
    FROM stages
    WHERE stages.stage_build_id = builds.build_id
    AND stages.stage_status IN ('pending', 'running')
)
ORDER BY build_id DESC
LIMIT 50;
`
const queryReposRunningStatus = `
SELECT
repo_namespace
,repo_name
,repo_slug
,build_number
,build_author
,build_author_name
,build_author_email
,build_author_avatar
,build_sender
,build_started
,build_finished
,build_created
,build_updated
,stage_name
,stage_kind
,stage_type
,stage_status
,stage_machine
,stage_os
,stage_arch
,stage_variant
,stage_kernel
,stage_limit
,stage_limit_repo
,stage_started
,stage_stopped
FROM repos
INNER JOIN builds ON builds.build_repo_id = repos.repo_id
inner join stages on stages.stage_build_id = builds.build_id
where stages.stage_status IN ('pending', 'running')
ORDER BY build_id DESC;
`
