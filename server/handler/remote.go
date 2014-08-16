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
	switch {
	case user == nil:
		return notAuthorized{}
	case user.Admin == false:
		return forbidden{}
	}
	// get all remotes
	remotes, err := h.remotes.List()
	if err != nil {
		return internalServerError{err}
	}

	return json.NewEncoder(w).Encode(remotes)
}

// GetRemoteLogins gets all remote logins.
// GET /api/remotes/logins
func (h *RemoteHandler) GetRemoteLogins(w http.ResponseWriter, r *http.Request) error {
	remotes, err := h.remotes.List()
	if err != nil {
		return internalServerError{err}
	}
	var logins []interface{}
	for _, remote := range remotes {
		logins = append(logins, struct {
			Type string `json:"type"`
			Host string `json:"host"`
		}{remote.Type, remote.Host})
	}
	return json.NewEncoder(w).Encode(&logins)
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
	// to add the same result multiple times. In this case we will delete
	// the old remote prior to adding the new one.
	if remote, err := h.remotes.FindHost(in.Host); err == nil && h.users.Exist() {
		h.remotes.Delete(remote)
	}

	// insert the remote in the database
	if err := h.remotes.Insert(&in); err != nil {
		return internalServerError{err}
	}

	return json.NewEncoder(w).Encode(&in)
}

// PutRemote updates an existing remote.
// PUT /api/remotes
func (h *RemoteHandler) PutRemote(w http.ResponseWriter, r *http.Request) error {
	// get the user form the session
	user := h.sess.User(r)
	switch {
	case user == nil:
		return notAuthorized{}
	case user.Admin == false:
		return forbidden{}
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

	// retrieve the remote and return an error if not exists
	remote, err := h.remotes.FindHost(in.Host)
	if err != nil {
		return notFound{err}
	}

	// update the remote details
	remote.API = in.API
	remote.URL = in.URL
	remote.Host = in.Host
	remote.Client = in.Client
	remote.Secret = in.Secret

	// insert the remote in the database
	if err := h.remotes.Update(remote); err != nil {
		return internalServerError{err}
	}

	return json.NewEncoder(w).Encode(remote)
}

func (h *RemoteHandler) Register(r *pat.Router) {
	r.Get("/v1/logins", errorHandler(h.GetRemoteLogins))
	r.Get("/v1/remotes", errorHandler(h.GetRemotes))
	r.Post("/v1/remotes", errorHandler(h.PostRemote))
	r.Put("/v1/remotes", errorHandler(h.PutRemote))
}
