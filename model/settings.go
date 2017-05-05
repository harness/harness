package model

// Settings defines system configuration parameters.
type Settings struct {
	Open   bool            // Enables open registration
	Secret string          // Secret token used to authenticate agents
	Admins map[string]bool // Administrative users
	Orgs   map[string]bool // Organization whitelist
}

// IsAdmin returns true if the user is a member of the administrator list.
func (c *Settings) IsAdmin(user *User) bool {
	return c.Admins[user.Login]
}

// IsMember returns true if the user is a member of the whitelisted teams.
func (c *Settings) IsMember(teams []*Team) bool {
	for _, team := range teams {
		if c.Orgs[team.Login] {
			return true
		}
	}
	return false
}
