// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package rpc2

import (
	"errors"
	"github.com/drone/drone/handler/api/render"
	"net/http"
	"strings"

	"github.com/drone/drone/cmd/drone-server/config"
	"github.com/drone/drone/operator/manager"

	"github.com/dchest/authcookie"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

// Server wraps the chi Router in a custom type for wire
// injection purposes.
type Server http.Handler

// NewServer returns a new rpc server that enables remote
// interaction with the build controller using the http transport.
func NewServer(manager manager.BuildManager, config config.Config) Server {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.NoCache)
	r.Use(authorization(config.RPC.Secret))
	r.Post("/nodes/:machine", HandleJoin())
	r.Delete("/nodes/:machine", HandleLeave())
	r.Post("/ping", HandlePing())
	r.Post("/stage", HandleRequest(manager))
	r.Post("/stage/{stage}", HandleAccept(manager))
	r.Get("/stage/{stage}", HandleInfo(manager))
	r.Put("/stage/{stage}", HandleUpdateStage(manager))
	r.Put("/step/{step}", HandleUpdateStep(manager))
	r.Post("/build/{build}/watch", HandleWatch(manager))
	r.Post("/step/{step}/logs/batch", HandleLogBatch(manager))
	r.Post("/step/{step}/logs/upload", HandleLogUpload(manager))
	r.Post("/cards", HandleCard(manager, config.Authn.InternalAuthSecret))
	return Server(r)
}

func HandleCard(manager manager.BuildManager, secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx   := r.Context()
		token := extractToken(r)
		s := []byte(secret)
		login := authcookie.Login(token, s)
		if login != "" {
			err := manager.HandleCard(ctx, r, login)
			if err != nil{
				render.BadRequest(w, err)
				return
			}
			writeOK(w)
		} else {
			writeError(w, errors.New("unauthorized"))
		}
	}
}

func extractToken(r *http.Request) string {
	bearer := r.Header.Get("Authorization")
	if bearer == "" {
		bearer = r.FormValue("access_token")
	}
	return strings.TrimPrefix(bearer, "Bearer ")
}

func authorization(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// prevents system administrators from accidentally
			// exposing drone without credentials.
			path := strings.Split(r.URL.Path, "?")
			if path[0] == "/rpc/v2/cards" {
				next.ServeHTTP(w, r)
			}
			if token == "" {
				w.WriteHeader(403)
			} else if token == r.Header.Get("X-Drone-Token") {
				next.ServeHTTP(w, r)
			} else {
				w.WriteHeader(401)
			}
		})
	}
}
