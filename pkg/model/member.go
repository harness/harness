package model

import (
	"fmt"
)

const (
	// Owners can add / remove team members, create / delete projects,
	// and have rwx access to all projects owned by the team.
	RoleOwner = "Owner"

	// Admins can create / delete projects and have rwx acess
	// to all projects owned by the team.
	RoleAdmin = "Admin"

	// Write members have rwx access to all projects
	// owned by the team. They may not create new projects.
	RoleWrite = "Write"

	// Read members have readonly access to all projects
	// owned by the team.
	RoleRead = "Read"
)

type Role struct {
	ID     int64  `meddler:"id,pk"`
	TeamID int64  `meddler:"team_id"`
	UserID int64  `meddler:"user_id"`
	Role   string `meddler:"role"`
}

type Member struct {
	ID       int64  `meddler:"user_id"`
	Name     string `meddler:"name"`
	Email    string `meddler:"email"`
	Gravatar string `meddler:"gravatar"`
	Role     string `meddler:"role"`
}

// Returns the Gravatar Image URL.
func (m *Member) Image() string      { return fmt.Sprintf(GravatarPattern, m.Gravatar, 42) }
func (m *Member) ImageSmall() string { return fmt.Sprintf(GravatarPattern, m.Gravatar, 32) }
func (m *Member) ImageLarge() string { return fmt.Sprintf(GravatarPattern, m.Gravatar, 160) }
