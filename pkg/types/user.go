package types

type User struct {
	ID     int64  `meddler:"user_id,pk"    json:"id"`
	Login  string `meddler:"user_login"    json:"login,omitempty" sql:"unique:ux_user_login"`
	Token  string `meddler:"user_token"    json:"-"`
	Secret string `meddler:"user_secret"   json:"-"`
	Email  string `meddler:"user_email"    json:"email,omitempty"`
	Avatar string `meddler:"user_gravatar" json:"gravatar_id,omitempty"`
	Active bool   `meddler:"user_active"   json:"active,omitempty"`
	Admin  bool   `meddler:"user_admin"    json:"admin,omitempty"`
}
