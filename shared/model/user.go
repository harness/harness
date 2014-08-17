package model

import (
	"time"
)

type User struct {
	Id       int64  `gorm:"primary_key:yes"     json:"-"`
	Remote   string `json:"remote"`
	Login    string `json:"login"`
	Access   string `json:"-"`
	Secret   string `json:"-"`
	Name     string `json:"name"`
	Email    string `json:"email,omitempty"`
	Gravatar string `json:"gravatar"`
	Token    string `json:"-"`
	Admin    bool   `json:"admin"`
	Active   bool   `json:"active"`
	Syncing  bool   `json:"syncing"`
	Created  int64  `json:"created_at"`
	Updated  int64  `json:"updated_at"`
	Synced   int64  `json:"synced_at"`
}

func NewUser(remote, login, email string) *User {
	user := User{}
	user.Token = generateToken()
	user.Login = login
	user.Remote = remote
	user.Active = true
	user.SetEmail(email)
	return &user
}

// SetEmail sets the email address and calculate the Gravatar hash.
func (u *User) SetEmail(email string) {
	u.Email = email
	u.Gravatar = createGravatar(email)
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
