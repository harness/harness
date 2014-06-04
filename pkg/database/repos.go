package database

import (
	"time"

	. "github.com/drone/drone/pkg/model"
	"github.com/russross/meddler"
)

// Name of the Repos table in the database
const repoTable = "repos"

// SQL Queries to retrieve a list of all repos belonging to a User.
const repoStmt = `
SELECT id, slug, host, owner, name, private, disabled, disabled_pr, scm, url, username, password,
public_key, private_key, params, timeout, privileged, created, updated, user_id, team_id
FROM repos
WHERE user_id = ? AND team_id = 0
ORDER BY slug ASC
`

// SQL Queries to retrieve a list of all repos belonging to a Team.
const repoTeamStmt = `
SELECT id, slug, host, owner, name, private, disabled, disabled_pr, scm, url, username, password,
public_key, private_key, params, timeout, privileged, created, updated, user_id, team_id
FROM repos
WHERE team_id = ?
ORDER BY slug ASC
`

// SQL Queries to retrieve a repo by id.
const repoFindStmt = `
SELECT id, slug, host, owner, name, private, disabled, disabled_pr, scm, url, username, password,
public_key, private_key, params, timeout, privileged, created, updated, user_id, team_id
FROM repos
WHERE id = ?
`

// SQL Queries to retrieve a repo by name.
const repoFindSlugStmt = `
SELECT id, slug, host, owner, name, private, disabled, disabled_pr, scm, url, username, password,
public_key, private_key, params, timeout, privileged, created, updated, user_id, team_id
FROM repos
WHERE slug = ?
`

// Returns the Repo with the given ID.
func GetRepo(id int64) (*Repo, error) {
	repo := Repo{}
	err := meddler.QueryRow(db, &repo, repoFindStmt, id)
	return &repo, err
}

// Returns the Repo with the given slug.
func GetRepoSlug(slug string) (*Repo, error) {
	repo := Repo{}
	err := meddler.QueryRow(db, &repo, repoFindSlugStmt, slug)
	return &repo, err
}

// Creates a new Repository.
func SaveRepo(repo *Repo) error {
	if repo.ID == 0 {
		repo.Created = time.Now().UTC()
	}
	repo.Updated = time.Now().UTC()
	return meddler.Save(db, repoTable, repo)
}

// Deletes an existing Repository.
// TODO need to delete builds too.
func DeleteRepo(id int64) error {
	_, err := db.Exec("DELETE FROM repos WHERE id = ?", id)
	db.Exec("DELETE FROM commits WHERE repo_id = ?", id)
	return err
}

// Returns a list of all Repos associated
// with the specified User ID.
func ListRepos(id int64) ([]*Repo, error) {
	var repos []*Repo
	err := meddler.QueryAll(db, &repos, repoStmt, id)
	return repos, err
}

// Returns a list of all Repos associated
// with the specified Team ID.
func ListReposTeam(id int64) ([]*Repo, error) {
	var repos []*Repo
	err := meddler.QueryAll(db, &repos, repoTeamStmt, id)
	return repos, err
}

// Checks whether a user is admin of a repo
// Returns true if user owns repo or is on team that owns repo
// Returns true if the user is an admin member of the team.
func IsRepoAdmin(user *User, repo *Repo) (bool, error) {
	if user == nil {
		return false, nil
	}

	if user.ID == repo.UserID {
		return true, nil
	}

	return IsMemberAdmin(user.ID, repo.TeamID)
}
