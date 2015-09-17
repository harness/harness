package types

type User struct {
	ID     int64  `json:"id"`
	Login  string `json:"login,omitempty" sql:"unique:ux_user_login"`
	Token  string `json:"token,omitempty"`
	Secret string `json:"-"`
	Email  string `json:"email,omitempty"`
	Avatar string `json:"avatar_url,omitempty"`
	Active bool   `json:"active,omitempty"`
	Admin  bool   `json:"admin,omitempty"`

	// randomly generated hash used to sign user
	// session and application tokens.
	Hash string `json:"-"`
}
