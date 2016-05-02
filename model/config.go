package model

// Config defines system configuration parameters.
type Config struct {
	Open   bool            // Enables open registration
	Yaml   string          // Customize the Yaml configuration file name
	Shasum string          // Customize the Yaml checksum file name
	Secret string          // Secret token used to authenticate agents
	Admins map[string]bool // Administrative users
	Orgs   map[string]bool // Organization whitelist
}

// IsAdmin returns true if the user is a member of the administrator list.
func (c *Config) IsAdmin(user *User) bool {
	return c.Admins[user.Login]
}

// IsMember returns true if the user is a member of the whitelisted teams.
func (c *Config) IsMember(teams []*Team) bool {
	for _, team := range teams {
		if c.Orgs[team.Login] {
			return true
		}
	}
	return false
}
