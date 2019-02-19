// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package users

import (
	"net/http"

	"github.com/drone/drone/handler/api/render"
	"github.com/drone/drone/logger"
	"github.com/drone/drone/core"

	"github.com/go-chi/chi"
)

// HandleDelete returns an http.HandlerFunc that processes an http.Request
// to delete the named user account from the system.
func HandleDelete(
	users core.UserStore,
	sender core.WebhookSender,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		login := chi.URLParam(r, "user")
		user, err := users.FindLogin(r.Context(), login)
		if err != nil {
			render.NotFound(w, err)
			logger.FromRequest(r).WithError(err).
				Debugln("api: cannot find user")
			return
		}

		err = users.Delete(r.Context(), user)
		if err != nil {
			render.InternalError(w, err)
			logger.FromRequest(r).WithError(err).
				Warnln("api: cannot delete user")
			return
		}

		err = sender.Send(r.Context(), &core.WebhookData{
			Event:  core.WebhookEventUser,
			Action: core.WebhookActionDeleted,
			User:   user,
		})
		if err != nil {
			logger.FromRequest(r).WithError(err).
				Warnln("api: cannot send webhook")
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
