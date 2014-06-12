package handler

import (
	"encoding/json"
	"net/http"

	"github.com/drone/drone/server/database"
	"github.com/drone/drone/server/session"
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
	if user == nil || !user.Admin {
		return notAuthorized{}
	}
	// get all users
	users, err := h.users.List()
	if err != nil {
		return internalServerError{err}
	}

	return json.NewEncoder(w).Encode(users)
}

func (h *UsersHandler) Register(r *pat.Router) {
	r.Get("/v1/users", errorHandler(h.GetUsers))
}
