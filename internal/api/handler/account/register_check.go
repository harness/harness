// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package account

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/user"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/types"
)

// HandleRegisterCheck returns an http.HandlerFunc that processes an http.Request
// and returns a boolean true/false if user registration is allowed or not
func HandleRegisterCheck(userCtrl *user.Controller, config *types.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		check, err := userCtrl.RegisterCheck(ctx, config)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}

		render.JSON(w, http.StatusOK, check)
	}
}
