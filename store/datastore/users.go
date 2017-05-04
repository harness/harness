package datastore

import (
	"fmt"

	"github.com/drone/drone/model"
	"github.com/drone/drone/store/datastore/sql"
	"github.com/russross/meddler"
)

func (db *datastore) GetUser(id int64) (*model.User, error) {
	var usr = new(model.User)
	var err = meddler.Load(db, userTable, usr, id)
	return usr, err
}

func (db *datastore) GetUserLogin(login string) (*model.User, error) {
	var usr = new(model.User)
	var err = meddler.QueryRow(db, usr, rebind(userLoginQuery), login)
	return usr, err
}

func (db *datastore) GetUserList() ([]*model.User, error) {
	var users = []*model.User{}
	var err = meddler.QueryAll(db, &users, rebind(userListQuery))
	return users, err
}

func (db *datastore) GetUserFeed(listof []*model.RepoLite) ([]*model.Feed, error) {
	var (
		args []interface{}
		stmt string
		err  error

		feed = []*model.Feed{}
	)
	switch meddler.Default {
	case meddler.PostgreSQL:
		stmt, args = toListPostgres(listof)
	default:
		stmt, args = toList(listof)
	}
	if len(args) > 0 {
		err = meddler.QueryAll(db, &feed, fmt.Sprintf(userFeedQuery, stmt), args...)
	}
	return feed, err
}

func (db *datastore) GetUserFeedLatest(listof []*model.RepoLite) ([]*model.Feed, error) {
	var (
		args []interface{}
		stmt string
		err  error

		feed = []*model.Feed{}
	)
	switch meddler.Default {
	case meddler.PostgreSQL:
		stmt, args = toListPostgres(listof)
	default:
		stmt, args = toList(listof)
	}
	if len(args) > 0 {
		err = meddler.QueryAll(db, &feed, fmt.Sprintf(userFeedLatest, stmt), args...)
	}
	return feed, err
}

func (db *datastore) GetUserCount() (count int, err error) {
	err = db.QueryRow(
		sql.Lookup(db.driver, "count-users"),
	).Scan(&count)
	return
}

func (db *datastore) CreateUser(user *model.User) error {
	return meddler.Insert(db, userTable, user)
}

func (db *datastore) UpdateUser(user *model.User) error {
	return meddler.Update(db, userTable, user)
}

func (db *datastore) DeleteUser(user *model.User) error {
	var _, err = db.Exec(rebind(userDeleteStmt), user.ID)
	return err
}

const userTable = "users"

const userLoginQuery = `
SELECT *
FROM users
WHERE user_login=?
LIMIT 1
`

const userListQuery = `
SELECT *
FROM users
ORDER BY user_login ASC
`

const userCountQuery = `
SELECT count(1)
FROM users
`

const userDeleteStmt = `
DELETE FROM users
WHERE user_id=?
`

const userFeedQuery = `
SELECT
 repo_owner
,repo_name
,repo_full_name
,build_number
,build_event
,build_status
,build_created
,build_started
,build_finished
,build_commit
,build_branch
,build_ref
,build_refspec
,build_remote
,build_title
,build_message
,build_author
,build_email
,build_avatar
FROM
 builds b
,repos r
WHERE b.build_repo_id = r.repo_id
  AND r.repo_full_name IN (%s)
ORDER BY b.build_id DESC
LIMIT 50
`

// thanks to this article for helping me find a sane sql query
// https://www.periscopedata.com/blog/4-ways-to-join-only-the-first-row-in-sql.html

const userFeedLatest = `
SELECT
 repo_owner
,repo_name
,repo_full_name
,build_number
,build_event
,build_status
,build_created
,build_started
,build_finished
,build_commit
,build_branch
,build_ref
,build_refspec
,build_remote
,build_title
,build_message
,build_author
,build_email
,build_avatar
FROM repos LEFT OUTER JOIN builds ON build_id = (
	SELECT build_id FROM builds
	WHERE builds.build_repo_id = repos.repo_id
	ORDER BY build_id DESC
	LIMIT 1
)
WHERE repo_full_name IN (%s)
`
