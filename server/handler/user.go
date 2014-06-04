package handler

import (
	"encoding/json"
	"net/http"

	"github.com/drone/drone/server/resource/commit"
	"github.com/drone/drone/server/resource/repo"
	"github.com/drone/drone/server/resource/user"
	"github.com/drone/drone/server/session"
	"github.com/gorilla/pat"
)

type UserHandler struct {
	commits commit.CommitManager
	repos   repo.RepoManager
	users   user.UserManager
	sess    session.Session
}

func NewUserHandler(users user.UserManager, repos repo.RepoManager, commits commit.CommitManager, sess session.Session) *UserHandler {
	return &UserHandler{commits, repos, users, sess}
}

// GetUser gets the authenticated user.
// GET /api/user
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) error {
	// get the user form the session
	u := h.sess.User(r)
	if u == nil {
		return notAuthorized{}
	}
	// Normally the Token would not be serialized to json.
	// In this case it is appropriate because the user is
	// requesting their own data, and will need to display
	// the Token on the website.
	data := struct {
		*user.User
		Token string `json:"token"`
	}{u, u.Token}
	return json.NewEncoder(w).Encode(&data)
}

// PutUser updates the authenticated user.
// PUT /api/user
func (h *UserHandler) PutUser(w http.ResponseWriter, r *http.Request) error {
	// get the user form the session
	u := h.sess.User(r)
	if u == nil {
		return notAuthorized{}
	}

	// unmarshal the repository from the payload
	defer r.Body.Close()
	in := user.User{}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		return badRequest{err}
	}

	// update the user email
	if len(in.Email) != 0 {
		u.SetEmail(in.Email)
	}
	// update the user full name
	if len(in.Name) != 0 {
		u.Name = in.Name
	}

	// update the database
	if err := h.users.Update(u); err != nil {
		return internalServerError{err}
	}

	return json.NewEncoder(w).Encode(u)
}

// GetRepos gets the authenticated user's repositories.
// GET /api/user/repos
func (h *UserHandler) GetRepos(w http.ResponseWriter, r *http.Request) error {
	// get the user from the session
	u := h.sess.User(r)
	if u == nil {
		return notAuthorized{}
	}

	// get the user repositories
	repos, err := h.repos.List(u.ID)
	if err != nil {
		return badRequest{err}
	}
	return json.NewEncoder(w).Encode(&repos)
}

// GetFeed gets the authenticated user's commit feed.
// GET /api/user/feed
func (h *UserHandler) GetFeed(w http.ResponseWriter, r *http.Request) error {
	// get the user from the session
	u := h.sess.User(r)
	if u == nil {
		return notAuthorized{}
	}

	// get the user commits
	commits, err := h.commits.ListUser(u.ID)
	if err != nil {
		return badRequest{err}
	}
	return json.NewEncoder(w).Encode(&commits)
}

func (h *UserHandler) Register(r *pat.Router) {
	r.Get("/v1/user/repos", errorHandler(h.GetRepos))
	r.Get("/v1/user/feed", errorHandler(h.GetFeed))
	r.Get("/v1/user", errorHandler(h.GetUser))
	r.Put("/v1/user", errorHandler(h.PutUser))
}
