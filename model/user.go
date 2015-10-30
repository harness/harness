package model

type User struct {
	ID     int64  `json:"id"         meddler:"user_id,pk"`
	Login  string `json:"login"      meddler:"user_login"`
	Token  string `json:"-"          meddler:"user_token"`
	Secret string `json:"-"          meddler:"user_secret"`
	Expiry int64  `json:"-"          meddler:"user_expiry"`
	Email  string `json:"email"      meddler:"user_email"`
	Avatar string `json:"avatar_url" meddler:"user_avatar"`
	Active bool   `json:"active,"    meddler:"user_active"`
	Admin  bool   `json:"admin,"     meddler:"user_admin"`
	Hash   string `json:"-"          meddler:"user_hash"`
}
