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

package batch2

import (
	"context"
	"fmt"
	"time"

	"github.com/drone/drone/core"
	"github.com/drone/drone/store/repos"
	"github.com/drone/drone/store/shared/db"
)

// New returns a new Batcher.
func New(db *db.DB) core.Batcher {
	return &batchUpdater{db}
}

type batchUpdater struct {
	db *db.DB
}

func (b *batchUpdater) Batch(ctx context.Context, user *core.User, batch *core.Batch) error {
	return b.db.Update(func(execer db.Execer, binder db.Binder) error {
		now := time.Now().Unix()

		//
		// the repository list API does not return permissions, which means we have
		// no way of knowing if permissions are current or not. We therefore mark all
		// permissions stale in the database, so that each one must be individually
		// verified at runtime.
		//

		stmt := permResetStmt
		switch b.db.Driver() {
		case db.Postgres:
			stmt = permResetStmtPostgres
		}

		_, err := execer.Exec(stmt, now, user.ID)
		if err != nil {
			return fmt.Errorf("batch: cannot reset permissions: %s", err)
		}

		// if the repository exists with the same name,
		// but a different unique identifier, attempt to
		// delete the previous entry.
		var insert []*core.Repository
		var update []*core.Repository
		for _, repo := range append(batch.Insert, batch.Update...) {
			params := repos.ToParams(repo)
			stmt, args, err := binder.BindNamed(repoDeleteDeleted, params)
			if err != nil {
				return err
			}
			res, err := execer.Exec(stmt, args...)
			if err != nil {
				return fmt.Errorf("batch: cannot remove duplicate repository: %s: %s: %s", repo.Slug, repo.UID, err)
			}
			rows, _ := res.RowsAffected()
			if rows > 0 {
				insert = append(insert, repo)
			} else if repo.ID > 0 {
				update = append(update, repo)
			} else {
				insert = append(insert, repo)
			}
		}

		for _, repo := range insert {

			//
			// insert repository
			// TODO: group inserts in batches of N
			//

			stmt := repoInsertIgnoreStmt
			switch b.db.Driver() {
			case db.Mysql:
				stmt = repoInsertIgnoreStmtMysql
			case db.Postgres:
				stmt = repoInsertIgnoreStmtPostgres
			}

			params := repos.ToParams(repo)
			stmt, args, err := binder.BindNamed(stmt, params)
			if err != nil {
				return err
			}
			_, err = execer.Exec(stmt, args...)
			if err != nil {
				return fmt.Errorf("batch: cannot insert repository: %s: %s: %s", repo.Slug, repo.UID, err)
			}

			//
			// insert permissions
			// TODO: group inserts in batches of N
			//

			stmt = permInsertIgnoreStmt
			switch b.db.Driver() {
			case db.Mysql:
				stmt = permInsertIgnoreStmtMysql
			case db.Postgres:
				stmt = permInsertIgnoreStmtPostgres
			}

			_, err = execer.Exec(stmt,
				user.ID,
				repo.UID,
				now,
				now,
			)
			if err != nil {
				return fmt.Errorf("batch: cannot insert permissions: %s: %s: %s", repo.Slug, repo.UID, err)
			}
		}

		//
		// update existing repositories
		// TODO: group updates in batches of N
		//

		for _, repo := range update {
			params := repos.ToParams(repo)

			// // if the repository exists with the same name,
			// // but a different unique identifier, attempt to
			// // delete the previous entry.
			// stmt, args, err := binder.BindNamed(repoDeleteDeleted, params)
			// if err != nil {
			// 	return err
			// }
			// res, err := execer.Exec(stmt, args...)
			// if err != nil {
			// 	return fmt.Errorf("batch: cannot remove duplicate repository: %s: %s: %s", repo.Slug, repo.UID, err)
			// }
			// rows, _ := res.RowsAffected()
			// if rows > 0 {
			// 	stmt := repoInsertIgnoreStmt
			// 	switch b.db.Driver() {
			// 	case db.Mysql:
			// 		stmt = repoInsertIgnoreStmtMysql
			// 	case db.Postgres:
			// 		stmt = repoInsertIgnoreStmtPostgres
			// 	}

			// 	params := repos.ToParams(repo)
			// 	stmt, args, err := binder.BindNamed(stmt, params)
			// 	if err != nil {
			// 		return err
			// 	}
			// 	_, err = execer.Exec(stmt, args...)
			// 	if err != nil {
			// 		return fmt.Errorf("batch: cannot insert repository: %s: %s: %s", repo.Slug, repo.UID, err)
			// 	}
			// } else {
			stmt, args, err := binder.BindNamed(repoUpdateRemoteStmt, params)
			if err != nil {
				return err
			}
			_, err = execer.Exec(stmt, args...)
			if err != nil {
				return fmt.Errorf("batch: cannot update repository: %s: %s: %s", repo.Slug, repo.UID, err)
			}
			// }

			stmt = permInsertIgnoreStmt
			switch b.db.Driver() {
			case db.Mysql:
				stmt = permInsertIgnoreStmtMysql
			case db.Postgres:
				stmt = permInsertIgnoreStmtPostgres
			}

			_, err = execer.Exec(stmt,
				user.ID,
				repo.UID,
				now,
				now,
			)
			if err != nil {
				return fmt.Errorf("batch: cannot insert permissions: %s: %s: %s", repo.Slug, repo.UID, err)
			}
		}

		//
		// revoke permissions
		// TODO: group deletes in batches of N
		//

		for _, repo := range batch.Revoke {
			stmt := permRevokeStmt
			switch b.db.Driver() {
			case db.Postgres:
				stmt = permRevokeStmtPostgres
			}

			_, err = execer.Exec(stmt, user.ID, repo.UID)
			if err != nil {
				return fmt.Errorf("batch: cannot revoking permissions: %s: %s: %s", repo.Slug, repo.UID, err)
			}
		}

		return nil
	})
}

const stmtInsertBase = `
(
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

const repoInsertIgnoreStmt = `
INSERT OR IGNORE INTO repos ` + stmtInsertBase

const repoInsertIgnoreStmtMysql = `
INSERT IGNORE INTO repos ` + stmtInsertBase

const repoInsertIgnoreStmtPostgres = `
INSERT INTO repos ` + stmtInsertBase + ` ON CONFLICT DO NOTHING`

const repoUpdateRemoteStmt = `
UPDATE repos SET
 repo_namespace=:repo_namespace
,repo_name=:repo_name
,repo_slug=:repo_slug
,repo_clone_url=:repo_clone_url
,repo_ssh_url=:repo_ssh_url
,repo_html_url=:repo_html_url
,repo_private=:repo_private
,repo_branch=:repo_branch
,repo_updated=:repo_updated
WHERE repo_id=:repo_id
`

const repoUpdateRemoteStmtPostgres = `
UPDATE repos SET
 repo_namespace=$1
,repo_name=$2
,repo_slug=$3
,repo_clone_url=$4
,repo_ssh_url=$5
,repo_html_url=$6
,repo_private=$7
,repo_branch=$8
,repo_updated=$9
WHERE repo_id=$10
`

const permInsertIgnoreStmt = `
INSERT OR IGNORE INTO perms (
 perm_user_id
,perm_repo_uid
,perm_read
,perm_write
,perm_admin
,perm_synced
,perm_created
,perm_updated
) values (
 ?
,?
,1
,0
,0
,0
,?
,?
)
`

const permInsertIgnoreStmtMysql = `
INSERT IGNORE INTO perms (
 perm_user_id
,perm_repo_uid
,perm_read
,perm_write
,perm_admin
,perm_synced
,perm_created
,perm_updated
) values (
 ?
,?
,1
,0
,0
,0
,?
,?
)
`

const permInsertIgnoreStmtPostgres = `
INSERT INTO perms (
 perm_user_id
,perm_repo_uid
,perm_read
,perm_write
,perm_admin
,perm_synced
,perm_created
,perm_updated
) values (
 $1
,$2
,true
,false
,false
,0
,$3
,$4
) ON CONFLICT DO NOTHING
`

// this statement deletes a repository that was
// deleted in version control and then re-created
// with the same name (and thus has a different
// unique identifier)
const repoDeleteDeleted = `
DELETE FROM repos
WHERE repo_slug = :repo_slug
  AND repo_uid != :repo_uid
`

// this resets the synced date indicating that
// the system should refresh the permissions next
// time the user attempts to access the resource
const permResetStmt = `
UPDATE perms SET
 perm_updated = ?
,perm_synced  = 0
WHERE perm_user_id = ?
`

const permResetStmtPostgres = `
UPDATE perms SET
 perm_updated = $1
,perm_synced  = 0
WHERE perm_user_id = $2
`

const permRevokeStmt = `
DELETE FROM perms
WHERE perm_user_id  = ?
AND   perm_repo_uid = ?
`

const permRevokeStmtPostgres = `
DELETE FROM perms
WHERE perm_user_id  = $1
AND   perm_repo_uid = $2
`
