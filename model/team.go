package model

// Team represents a team or organization in the remote version control system.
//
// swagger:model user
type Team struct {
	// Login is the username for this team.
	Login string `json:"login"`

	// the avatar url for this team.
	Avatar string `json:"avatar_url"`
}
