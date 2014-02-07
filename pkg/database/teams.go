package database

import (
	"time"

	. "github.com/drone/drone/pkg/model"
	"github.com/russross/meddler"
)

// Name of the Team table in the database
const teamTable = "teams"

// SQL Queries to retrieve a list of all teams belonging to a user.
const teamStmt = `
SELECT id, slug, name, email, gravatar, created, updated
FROM teams
WHERE id IN (select team_id from members where user_id = ?)
`

// SQL Queries to retrieve a team by id and user.
const teamFindStmt = `
SELECT id, slug, name, email, gravatar, created, updated
FROM teams
WHERE id = ?
`

// SQL Queries to retrieve a team by slug.
const teamFindSlugStmt = `
SELECT id, slug, name, email, gravatar, created, updated
FROM teams
WHERE slug = ?
`

// Returns the Team with the given ID.
func GetTeam(id int64) (*Team, error) {
	team := Team{}
	err := meddler.QueryRow(db, &team, teamFindStmt, id)
	return &team, err
}

// Returns the Team with the given slug.
func GetTeamSlug(slug string) (*Team, error) {
	team := Team{}
	err := meddler.QueryRow(db, &team, teamFindSlugStmt, slug)
	return &team, err
}

// Saves a Team.
func SaveTeam(team *Team) error {
	if team.ID == 0 {
		team.Created = time.Now().UTC()
	}
	team.Updated = time.Now().UTC()
	return meddler.Save(db, teamTable, team)
}

// Deletes an existing Team account.
func DeleteTeam(id int64) error {
	// disassociate all repos with this team
	db.Exec("UPDATE repos SET team_id = 0 WHERE team_id = ?", id)
	// delete the team memberships and the team itself
	db.Exec("DELETE FROM members WHERE team_id = ?", id)
	db.Exec("DELETE FROM teams WHERE id = ?", id)
	return nil
}

// Returns a list of all Teams associated
// with the specified User ID.
func ListTeams(id int64) ([]*Team, error) {
	var teams []*Team
	err := meddler.QueryAll(db, &teams, teamStmt, id)
	return teams, err
}
