package user

import (
	"github.com/drone/drone/server/resource/util"
	"time"
)

type User struct {
	ID       int64  `meddler:"user_id,pk"     json:"-"`
	ParentID int64  `meddler:"user_parent_id" json:"-"`
	Remote   string `meddler:"user_remote"    json:"remote"`
	Login    string `meddler:"user_login"     json:"login"`
	Access   string `meddler:"user_access"    json:"-"`
	Secret   string `meddler:"user_secret"    json:"-"`
	Name     string `meddler:"user_name"      json:"name"`
	Email    string `meddler:"user_email"     json:"email,omitempty"`
	Gravatar string `meddler:"user_gravatar"  json:"gravatar"`
	Token    string `meddler:"user_token"     json:"-"`
	Admin    bool   `meddler:"user_admin"     json:"admin"`
	Active   bool   `meddler:"user_active"    json:"active"`
	Created  int64  `meddler:"user_created"   json:"created_at"`
	Updated  int64  `meddler:"user_updated"   json:"updated_at"`
	Synced   int64  `meddler:"user_synced"    json:"synced_at"`
}

func New(remote, login, email string) *User {
	user := User{}
	user.Token = util.GenerateToken()
	user.Login = login
	user.Remote = remote
	user.Active = true
	user.SetEmail(email)
	return &user
}

// SetEmail sets the email address and calculate the Gravatar hash.
func (u *User) SetEmail(email string) {
	u.Email = email
	u.Gravatar = util.CreateGravatar(email)
}

func (u *User) Stale() bool {
	switch {
	case u.Synced == 0:
		return true
	// refresh every 24 hours
	case u.Synced+expires < time.Now().Unix():
		return true
	default:
		return false
	}
}

// by default, let's expire the user
// cache after 72 hours
var expires = int64(time.Hour.Seconds() * 72)
