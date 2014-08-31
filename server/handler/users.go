package handler

import (
	"encoding/json"
	"net/http"

	"github.com/drone/drone/server/database"
	"github.com/drone/drone/server/session"
	"github.com/drone/drone/shared/model"
	"github.com/gorilla/pat"
)

type UsersHandler struct {
	users database.UserManager
	sess  session.Session
}

func NewUsersHandler(users database.UserManager, sess session.Session) *UsersHandler {
	return &UsersHandler{users, sess}
}

// GetUsers gets all users.
// GET /api/users
func (h *UsersHandler) GetUsers(w http.ResponseWriter, r *http.Request) error {
	// get the user form the session
	user := h.sess.User(r)
	switch {
	case user == nil:
		return notAuthorized{}
	case user.Admin == false:
		return forbidden{}
	}
	// get all users
	users, err := h.users.List()
	if err != nil {
		return internalServerError{err}
	}

	return json.NewEncoder(w).Encode(users)
}

// GetUser gets a user by hostname and login.
// GET /api/users/:host/:login
func (h *UsersHandler) GetUser(w http.ResponseWriter, r *http.Request) error {
	remote := r.FormValue(":host")
	login := r.FormValue(":login")

	// get the user form the session
	user := h.sess.User(r)
	switch {
	case user == nil:
		return notAuthorized{}
	case user.Admin == false:
		return forbidden{}
	}
	user, err := h.users.FindLogin(remote, login)
	if err != nil {
		return notFound{err}
	}

	return json.NewEncoder(w).Encode(user)
}

// PostUser registers a new user account.
// POST /api/users/:host/:login
func (h *UsersHandler) PostUser(w http.ResponseWriter, r *http.Request) error {
	remote := r.FormValue(":host")
	login := r.FormValue(":login")

	// get the user form the session
	user := h.sess.User(r)
	switch {
	case user == nil:
		return notAuthorized{}
	case user.Admin == false:
		return forbidden{}
	}

	account := model.NewUser(remote, login, "")
	if err := h.users.Insert(account); err != nil {
		return badRequest{err}
	}

	return json.NewEncoder(w).Encode(account)
}

// DeleteUser gets a user by hostname and login and deletes
// from the system.
//
// DELETE /api/users/:host/:login
func (h *UsersHandler) DeleteUser(w http.ResponseWriter, r *http.Request) error {
	remote := r.FormValue(":host")
	login := r.FormValue(":login")

	// get the user form the session
	user := h.sess.User(r)
	switch {
	case user == nil:
		return notAuthorized{}
	case user.Admin == false:
		return forbidden{}
	}
	account, err := h.users.FindLogin(remote, login)
	if err != nil {
		return notFound{err}
	}

	// user cannot delete his / her own account
	if account.Id == user.Id {
		return badRequest{}
	}

	if err := h.users.Delete(account); err != nil {
		return badRequest{err}
	}

	// return a 200 indicating deletion complete
	w.WriteHeader(http.StatusOK)
	return nil
}

func (h *UsersHandler) Register(r *pat.Router) {
	r.Delete("/v1/users/{host}/{login}", errorHandler(h.DeleteUser))
	r.Post("/v1/users/{host}/{login}", errorHandler(h.PostUser))
	r.Get("/v1/users/{host}/{login}", errorHandler(h.GetUser))
	r.Get("/v1/users", errorHandler(h.GetUsers))
}
