package common

type User struct {
	Login    string `json:"login,omitempty"`
	Token    string `json:"-"`
	Secret   string `json:"-"`
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
	Gravatar string `json:"gravatar_id,omitempty"`
	Admin    bool   `json:"admin,omitempty"`
	Created  int64  `json:"created_at,omitempty"`
	Updated  int64  `json:"updated_at,omitempty"`

	// Repos contains a list of subscriptions
	// to repositories the user is watching.
	Repos map[string]struct{} `json:"-"`

	// Tokens contains a list of tokens for
	// the user account.
	Tokens map[string]struct{} `json:"-"`
}
