package session

import (
	"net/http"

	"github.com/drone/drone/server/resource/user"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

// stores sessions using secure cookies.
var cookies = sessions.NewCookieStore(
	securecookie.GenerateRandomKey(64))

type Session interface {
	User(r *http.Request) *user.User
	UserToken(r *http.Request) *user.User
	UserCookie(r *http.Request) *user.User
	SetUser(w http.ResponseWriter, r *http.Request, u *user.User)
	Clear(w http.ResponseWriter, r *http.Request)
}

type session struct {
	users user.UserManager
}

func NewSession(users user.UserManager) Session {
	return &session{
		users: users,
	}
}

// User gets the currently authenticated user from the secure cookie session.
func (s *session) User(r *http.Request) *user.User {
	//if true {
	//	user, _ := s.users.Find(1)
	//	return user
	//}

	switch {
	case r.FormValue("access_token") == "":
		return s.UserCookie(r)
	case r.FormValue("access_token") != "":
		return s.UserToken(r)
	}
	return nil
}

// UserToken gets the currently authenticated user for the given auth token.
func (s *session) UserToken(r *http.Request) *user.User {
	token := r.FormValue("access_token")
	user, _ := s.users.FindToken(token)
	return user
}

// UserCookie gets the currently authenticated user from the secure cookie session.
func (s *session) UserCookie(r *http.Request) *user.User {
	sess, err := cookies.Get(r, "_sess")
	if err != nil {
		return nil
	}
	// get the uid from the session
	value, ok := sess.Values["uid"]
	if !ok {
		return nil
	}
	// get the user from the database
	user, _ := s.users.Find(value.(int64))
	return user
}

// SetUser writes the specified username to the session.
func (s *session) SetUser(w http.ResponseWriter, r *http.Request, u *user.User) {
	sess, _ := cookies.Get(r, "_sess")
	sess.Values["uid"] = u.ID
	sess.Save(r, w)
}

// Clear removes the user from the session.
func (s *session) Clear(w http.ResponseWriter, r *http.Request) {
	sess, _ := cookies.Get(r, "_sess")
	delete(sess.Values, "uid")
	sess.Save(r, w)
}
