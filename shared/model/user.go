package model

import (
	"time"
)

type User struct {
	ID          int64  `meddler:"user_id,pk"          json:"-"`
	Remote      string `meddler:"user_remote"         json:"remote"`
	Login       string `meddler:"user_login"          json:"login"`
	Access      string `meddler:"user_access"         json:"-"`
	Secret      string `meddler:"user_secret"         json:"-"`
	Name        string `meddler:"user_name"           json:"name"`
	Email       string `meddler:"user_email"          json:"email,omitempty"`
	Gravatar    string `meddler:"user_gravatar"       json:"gravatar"`
	Token       string `meddler:"user_token"          json:"-"`
	Admin       bool   `meddler:"user_admin"          json:"admin"`
	Active      bool   `meddler:"user_active"         json:"active"`
	Syncing     bool   `meddler:"user_syncing"        json:"syncing"`
	Created     int64  `meddler:"user_created"        json:"created_at"`
	Updated     int64  `meddler:"user_updated"        json:"updated_at"`
	Synced      int64  `meddler:"user_synced"         json:"synced_at"`
	TokenExpiry int64  `meddler:"user_access_expires,zeroisnull" json:"-"`
}

func NewUser(remote, login, email string) *User {
	user := User{}
	user.Token = GenerateToken()
	user.Login = login
	user.Remote = remote
	user.Active = true
	user.SetEmail(email)
	return &user
}

// SetEmail sets the email address and calculate the Gravatar hash.
func (u *User) SetEmail(email string) {
	u.Email = email
	u.Gravatar = CreateGravatar(email)
}

func (u *User) IsStale() bool {
	switch {
	case u.Synced == 0:
		return true
	// refresh every 24 hours
	case u.Synced+DefaultExpires < time.Now().Unix():
		return true
	default:
		return false
	}
}

// by default, let's expire the user
// cache after 72 hours
var DefaultExpires = int64(time.Hour.Seconds() * 72)
