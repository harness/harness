package model

import (
	"errors"
	"fmt"
	"time"
)

// ErrInvalidTeamName is returned by the Team validation function
// when a team name is invalid.
var ErrInvalidTeamName = errors.New("Invalid Team Name")

type Team struct {
	ID       int64     `meddler:"id,pk"           json:"id"`
	Slug     string    `meddler:"slug"            json:"slug"`
	Name     string    `meddler:"name"            json:"name"`
	Email    string    `meddler:"email"           json:"email"`
	Gravatar string    `meddler:"gravatar"        json:"gravatar"`
	Created  time.Time `meddler:"created,utctime" json:"created"`
	Updated  time.Time `meddler:"updated,utctime" json:"updated"`
}

// Creates a new team with the specified email address,
// and team name.
func NewTeam(name, email string) *Team {
	team := Team{}
	team.SetEmail(email)
	team.SetName(name)
	return &team
}

// Returns the Gravatar Image URL.
func (t *Team) Image() string      { return fmt.Sprintf(GravatarPattern, t.Gravatar, 42) }
func (t *Team) ImageSmall() string { return fmt.Sprintf(GravatarPattern, t.Gravatar, 32) }
func (t *Team) ImageLarge() string { return fmt.Sprintf(GravatarPattern, t.Gravatar, 160) }

// Set the name and calculate the slug value.
func (t *Team) SetName(name string) {
	t.Name = name
	t.Slug = createSlug(name)
}

// Set the email address and calculate the
// Gravatar hash.
func (t *Team) SetEmail(email string) {
	t.Email = email
	t.Gravatar = createGravatar(email)
}

// ValidatePassword will compares the supplied password to
// the user password stored in the database.
func (t *Team) Validate() error {
	switch {
	case len(t.Slug) == 0:
		return ErrInvalidTeamName
	case len(t.Slug) >= 255:
		return ErrInvalidTeamName
	case len(t.Email) == 0:
		return ErrInvalidEmail
	case len(t.Email) >= 255:
		return ErrInvalidEmail
	case RegexpEmail.MatchString(t.Email) == false:
		return ErrInvalidEmail
	default:
		return nil
	}
}
