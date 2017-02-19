package datastore

import (
	"fmt"
	"sort"

	"github.com/drone/drone/model"
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
		err error
		feed = feedHelper{}
		total = len(listof)
		toListRepoLite func([]*model.RepoLite) (string, []interface{})
	)

	if total == 0 {
		return feed, nil
	}

	switch meddler.Default {
	case meddler.PostgreSQL:
		toListRepoLite = toListPosgres
	default:
		toListRepoLite = toList
	}

	pages := calculatePagination(total, maxRepoPage)
	for i := 0; i < pages; i++ {
		stmt, args := toListRepoLite(resizeList(listof, i, maxRepoPage))

		var tmpFeed []*model.Feed
		err = meddler.QueryAll(db, &tmpFeed, fmt.Sprintf(userFeedQuery, stmt), args...)
		if err != nil {
			return nil, err
		}

		feed = append(feed, tmpFeed...)
	}

	if len(feed) <= 50 {
		return feed, nil
	}

	sort.Sort(sort.Reverse(feed))

	return feed[:50], nil
}

func (db *datastore) GetUserFeedLatest(listof []*model.RepoLite) ([]*model.Feed, error) {
	var (
		err error
		feed = []*model.Feed{}
		toListRepoLite func([]*model.RepoLite) (string, []interface{})
		total = len(listof)
	)

	if total == 0 {
		return feed, nil
	}

	switch meddler.Default {
	case meddler.PostgreSQL:
		toListRepoLite = toListPosgres
	default:
		toListRepoLite = toList
	}

	pages := calculatePagination(total, maxRepoPage)
	for i := 0; i < pages; i++ {
		stmt, args := toListRepoLite(resizeList(listof, i, maxRepoPage))
		var tmpFeed []*model.Feed
		err = meddler.QueryAll(db, &tmpFeed, fmt.Sprintf(userFeedLatest, stmt), args...)
		if err != nil {
			return nil, err
		}

		feed = append(feed, tmpFeed...)
	}
	return feed, nil
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