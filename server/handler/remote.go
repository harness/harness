package handler

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/drone/drone/server/database"
	"github.com/drone/drone/server/session"
	"github.com/drone/drone/shared/model"
	"github.com/gorilla/pat"
)

type RemoteHandler struct {
	users   database.UserManager
	remotes database.RemoteManager
	sess    session.Session
}

func NewRemoteHandler(users database.UserManager, remotes database.RemoteManager, sess session.Session) *RemoteHandler {
	return &RemoteHandler{users, remotes, sess}
}

// GetRemotes gets all remotes.
// GET /api/remotes
func (h *RemoteHandler) GetRemotes(w http.ResponseWriter, r *http.Request) error {
	// get the user form the session
	user := h.sess.User(r)
	if user == nil || !user.Admin {
		return notAuthorized{}
	}
	// get all remotes
	remotes, err := h.remotes.List()
	if err != nil {
		return internalServerError{err}
	}

	return json.NewEncoder(w).Encode(remotes)
}

// PostRemote creates a new remote.
// POST /api/remotes
func (h *RemoteHandler) PostRemote(w http.ResponseWriter, r *http.Request) error {
	// get the user form the session
	user := h.sess.User(r)
	if user == nil || !user.Admin {
		// if no users exist, this request is part of
		// the system installation process and can proceed.
		// else we should reject.
		if h.users.Exist() {
			return notAuthorized{}
		}
	}
	// unmarshal the remote from the payload
	defer r.Body.Close()
	in := model.Remote{}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		return badRequest{err}
	}

	uri, err := url.Parse(in.URL)
	if err != nil {
		return badRequest{err}
	}
	in.Host = uri.Host

	// there is an edge case where, during installation, a user could attempt
	// to add the same result multiple times.
	if remote, err := h.remotes.FindHost(in.Host); err == nil && h.users.Exist() {
		h.remotes.Delete(remote)
	}

	// insert the remote in the database
	if err := h.remotes.Insert(&in); err != nil {
		return internalServerError{err}
	}

	return json.NewEncoder(w).Encode(&in)
}

// DeleteRemote delete the remote.
// GET /api/remotes/:name
func (h *RemoteHandler) DeleteRemote(w http.ResponseWriter, r *http.Request) error {
	host := r.FormValue(":host")

	// get the user form the session
	user := h.sess.User(r)
	if user == nil || !user.Admin {
		return notAuthorized{}
	}
	// get the remote
	remote, err := h.remotes.FindHost(host)
	if err != nil {
		return notFound{err}
	}
	if err := h.remotes.Delete(remote); err != nil {
		return internalServerError{err}
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (h *RemoteHandler) Register(r *pat.Router) {
	r.Delete("/v1/remotes/:name", errorHandler(h.DeleteRemote))
	r.Post("/v1/remotes", errorHandler(h.PostRemote))
	r.Get("/v1/remotes", errorHandler(h.GetRemotes))
}
