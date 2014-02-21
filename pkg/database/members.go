package database

import (
	. "github.com/drone/drone/pkg/model"
	"github.com/russross/meddler"
)

// Name of the Member table in the database
const memberTable = "members"

// SQL Queries to retrieve a list of all members belonging to a team.
const memberStmt = `
SELECT user_id, name, email, gravatar, role
FROM members, users
WHERE users.id = members.user_id
AND   team_id = ?
`

// SQL Queries to retrieve a team by id and user.
const memberFindStmt = `
SELECT user_id, name, email, gravatar, role
FROM members, users
WHERE users.id = members.user_id
AND   user_id = ?
AND   team_id = ?
`

// SQL Queries to retrieve a team by name .
const memberDeleteStmt = `
DELETE FROM members
WHERE user_id = ? AND team_id = ?
`

// SQL Queries to retrieve a member's role by id and user.
const roleFindStmt = `
SELECT id, team_id, user_id, role FROM members
WHERE user_id = ? AND team_id = ?
`

// Returns the Member with the given user and team IDs.
func GetMember(user, team int64) (*Member, error) {
	member := Member{}
	err := meddler.QueryRow(db, &member, memberFindStmt, user, team)
	return &member, err
}

// Returns true if the user is a member of the team
func IsMember(user, team int64) (bool, error) {
	role := Role{}
	err := meddler.QueryRow(db, &role, roleFindStmt, user, team)
	return len(role.Role) > 0, err
}

// Returns true is the user is an admin member of the team.
func IsMemberAdmin(user, team int64) (bool, error) {
	role := Role{}
	err := meddler.QueryRow(db, &role, roleFindStmt, user, team)
	return role.Role == RoleAdmin || role.Role == RoleOwner, err
}

// Creates a new Member.
func SaveMember(user, team int64, role string) error {
	r := Role{}
	if err := meddler.QueryRow(db, &r, roleFindStmt, user, team); err == nil {
		r.Role = role
		return meddler.Save(db, memberTable, &r)
	}

	r.UserID = user
	r.TeamID = team
	r.Role = role
	return meddler.Save(db, memberTable, &r)
}

// Deletes an existing Member.
func DeleteMember(user, team int64) error {
	_, err := db.Exec(memberDeleteStmt, user, team)
	return err
}

// Returns a list of all Team members.
func ListMembers(team int64) ([]*Member, error) {
	var members []*Member
	err := meddler.QueryAll(db, &members, memberStmt, team)
	return members, err
}
