package types

type User struct {
	ID     int64  `json:"id"`
	Login  string `json:"login,omitempty" sql:"unique:ux_user_login"`
	Token  string `json:"-"`
	Secret string `json:"-"`
	Email  string `json:"email,omitempty"`
	Avatar string `json:"avatar,omitempty"`
	Active bool   `json:"active,omitempty"`
	Admin  bool   `json:"admin,omitempty"`
}
