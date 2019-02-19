// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package user

import (
	"net/http"

	"github.com/dchest/uniuri"
	"github.com/drone/drone/handler/api/render"
	"github.com/drone/drone/handler/api/request"
	"github.com/drone/drone/core"
)

type userWithToken struct {
	*core.User
	Token string `json:"token"`
}

// HandleToken returns an http.HandlerFunc that writes json-encoded
// account information to the http response body with the user token.
func HandleToken(users core.UserStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		viewer, _ := request.UserFrom(ctx)
		if r.FormValue("rotate") == "true" {
			viewer.Hash = uniuri.NewLen(32)
			if err := users.Update(ctx, viewer); err != nil {
				render.InternalError(w, err)
				return
			}
		}
		render.JSON(w, &userWithToken{viewer, viewer.Hash}, 200)
	}
}
