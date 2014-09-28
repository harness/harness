package handler

import (
	"encoding/json"
	"net/http"

	"github.com/drone/drone/plugin/remote"
	"github.com/drone/drone/server/database"
	"github.com/drone/drone/server/session"
	"github.com/gorilla/pat"
)

type RemoteHandler struct {
	users database.UserManager
	sess  session.Session
}

func NewRemoteHandler(users database.UserManager, sess session.Session) *RemoteHandler {
	return &RemoteHandler{users, sess}
}

// GetRemoteLogins gets all remote logins.
// GET /api/remotes/logins
func (h *RemoteHandler) GetRemoteLogins(w http.ResponseWriter, r *http.Request) error {
	var list = remote.Registered()
	var logins []interface{}
	for _, item := range list {
		logins = append(logins, struct {
			Type string `json:"type"`
			Host string `json:"host"`
		}{item.GetKind(), item.GetHost()})
	}
	return json.NewEncoder(w).Encode(&logins)
}

func (h *RemoteHandler) Register(r *pat.Router) {
	r.Get("/v1/logins", errorHandler(h.GetRemoteLogins))
}
