package handler

import (
	"encoding/json"
	"net/http"

	"github.com/drone/drone/server/database"
	"github.com/drone/drone/server/session"
	"github.com/drone/drone/shared/model"
	"github.com/gorilla/pat"
)

type ServerHandler struct {
	servers database.ServerManager
	sess    session.Session
}

func NewServerHandler(servers database.ServerManager, sess session.Session) *ServerHandler {
	return &ServerHandler{servers, sess}
}

// GetServers gets all servers.
// GET /api/servers
func (h *ServerHandler) GetServers(w http.ResponseWriter, r *http.Request) error {
	// get the user form the session
	user := h.sess.User(r)
	if user == nil || !user.Admin {
		return notAuthorized{}
	}
	// get all servers
	servers, err := h.servers.List()
	if err != nil {
		return internalServerError{err}
	}

	return json.NewEncoder(w).Encode(servers)
}

// PostServer creates a new server.
// POST /api/servers
func (h *ServerHandler) PostServer(w http.ResponseWriter, r *http.Request) error {
	// get the user form the session
	user := h.sess.User(r)
	if user == nil || !user.Admin {
		return notAuthorized{}
	}
	// unmarshal the server from the payload
	defer r.Body.Close()
	in := model.Server{}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		return badRequest{err}
	}
	// insert the server in the database
	if err := h.servers.Insert(&in); err != nil {
		return internalServerError{err}
	}

	return json.NewEncoder(w).Encode(&in)
}

// DeleteServers deletes the named server.
// GET /api/servers/:name
func (h *ServerHandler) DeleteServer(w http.ResponseWriter, r *http.Request) error {
	name := r.FormValue(":name")

	// get the user form the session
	user := h.sess.User(r)
	if user == nil || !user.Admin {
		return notAuthorized{}
	}
	// get the server
	server, err := h.servers.FindName(name)
	if err != nil {
		return notFound{err}
	}
	if err := h.servers.Delete(server); err != nil {
		return internalServerError{err}
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (h *ServerHandler) Register(r *pat.Router) {
	r.Delete("/v1/servers/:name", errorHandler(h.DeleteServer))
	r.Post("/v1/servers", errorHandler(h.PostServer))
	r.Get("/v1/servers", errorHandler(h.GetServers))
}
