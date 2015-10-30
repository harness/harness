package datastore

import (
	"database/sql"
	"fmt"

	"github.com/drone/drone/model"
	"github.com/russross/meddler"
)

type userstore struct {
	*sql.DB
}

func (db *userstore) Get(id int64) (*model.User, error) {
	var usr = new(model.User)
	var err = meddler.Load(db, userTable, usr, id)
	return usr, err
}

func (db *userstore) GetLogin(login string) (*model.User, error) {
	var usr = new(model.User)
	var err = meddler.QueryRow(db, usr, rebind(userLoginQuery), login)
	return usr, err
}

func (db *userstore) GetList() ([]*model.User, error) {
	var users = []*model.User{}
	var err = meddler.QueryAll(db, &users, rebind(userListQuery))
	return users, err
}

func (db *userstore) GetFeed(listof []*model.RepoLite) ([]*model.Feed, error) {
	var (
		feed []*model.Feed
		args []interface{}
		stmt string
	)
	switch meddler.Default {
	case meddler.PostgreSQL:
		stmt, args = toListPosgres(listof)
	default:
		stmt, args = toList(listof)
	}
	err := meddler.QueryAll(db, &feed, fmt.Sprintf(userFeedQuery, stmt), args...)
	return feed, err
}

func (db *userstore) Count() (int, error) {
	var count int
	var err = db.QueryRow(rebind(userCountQuery)).Scan(&count)
	return count, err
}

func (db *userstore) Create(user *model.User) error {
	return meddler.Insert(db, userTable, user)
}

func (db *userstore) Update(user *model.User) error {
	return meddler.Update(db, userTable, user)
}

func (db *userstore) Delete(user *model.User) error {
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
LIMIT 25
`
