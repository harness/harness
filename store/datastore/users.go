package datastore

import (
	"fmt"

	"github.com/drone/drone/model"
	"github.com/russross/meddler"
	"sort"
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
		err  error
		feed = model.Feeds{}
	)
	_stmt, _args := toList(listof)

	for i, stmt := range _stmt{
		args = _args[i]
		if len(args) > 0 {
			var _feed []*model.Feed
			err = meddler.QueryAll(db, &_feed, fmt.Sprintf(userFeedQuery, stmt), args...)
			if err != nil{
				break
			}
			feed = append(feed, _feed...)
		}
	}
	sort.Sort(sort.Reverse(feed))
	limit := 50
	if len(feed) < limit{
		limit = len(feed)
	}
	return feed[:limit], err
}

func (db *datastore) GetUserFeedLatest(listof []*model.RepoLite) ([]*model.Feed, error) {
	var (
		args []interface{}
		err  error

		feed = []*model.Feed{}
	)
	_stmt, _args := toList(listof)

	for i, stmt := range(_stmt){
		args = _args[i]
		if len(args) > 0 {
			var _feed []*model.Feed
			err = meddler.QueryAll(db, &_feed, fmt.Sprintf(userFeedLatest, stmt), args...)
			if err != nil{
				break
			}
			feed = append(feed, _feed...)
		}
	}
	return feed, err
}

func (db *datastore) GetUserCount() (int, error) {
	var count int
	var err = db.QueryRow(rebind(userCountQuery)).Scan(&count)
	return count, err
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
,build_id
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
