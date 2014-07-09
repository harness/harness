package session

import (
	"net/http"

	"github.com/drone/drone/server/database"
	"github.com/drone/drone/shared/model"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

// stores sessions using secure cookies.
var cookies = sessions.NewCookieStore(
	securecookie.GenerateRandomKey(64))

type Session interface {
	User(r *http.Request) *model.User
	UserToken(r *http.Request) *model.User
	UserCookie(r *http.Request) *model.User
	SetUser(w http.ResponseWriter, r *http.Request, u *model.User)
	Clear(w http.ResponseWriter, r *http.Request)
}

type session struct {
	users database.UserManager
}

func NewSession(users database.UserManager) Session {
	return &session{
		users: users,
	}
}

// User gets the currently authenticated user from the secure cookie session.
func (s *session) User(r *http.Request) *model.User {
	switch {
	case r.FormValue("access_token") == "":
		return s.UserCookie(r)
	case r.FormValue("access_token") != "":
		return s.UserToken(r)
	}
	return nil
}

// UserToken gets the currently authenticated user for the given auth token.
func (s *session) UserToken(r *http.Request) *model.User {
	token := r.FormValue("access_token")
	user, _ := s.users.FindToken(token)
	return user
}

// UserCookie gets the currently authenticated user from the secure cookie session.
func (s *session) UserCookie(r *http.Request) *model.User {
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
func (s *session) SetUser(w http.ResponseWriter, r *http.Request, u *model.User) {
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
