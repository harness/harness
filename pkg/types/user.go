package types

type User struct {
	ID       int64  `meddler:"user_id,pk"    json:"-"`
	Login    string `meddler:"user_login"    json:"login,omitempty"`
	Token    string `meddler:"user_token"    json:"-"`
	Secret   string `meddler:"user_secret"   json:"-"`
	Name     string `meddler:"user_name"     json:"name,omitempty"`
	Email    string `meddler:"user_email"    json:"email,omitempty"`
	Gravatar string `meddler:"user_gravatar" json:"gravatar_id,omitempty"`
	Admin    bool   `meddler:"user_admin"    json:"admin,omitempty"`
	Active   bool   `meddler:"user_active"   json:"active,omitempty"`
	Created  int64  `meddler:"user_created"  json:"created_at,omitempty"`
	Updated  int64  `meddler:"user_updated"  json:"updated_at,omitempty"`
}
