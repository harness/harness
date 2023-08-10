// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package system

import (
	"net/http"

	"github.com/harness/gitness/internal/api/controller/system"
	"github.com/harness/gitness/internal/api/render"
)

type ConfigOutput struct {
	SignUpAllowed bool `json:"sign_up_allowed"`
}

// HandleGetConfig returns an http.HandlerFunc that processes an http.Request
// and returns a struct containing all system configs exposed to the users.
func HandleGetConfig(sysCtrl *system.Controller) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		signUpAllowedCheck, err := sysCtrl.RegisterCheck(ctx)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}
		render.JSON(w, http.StatusOK, ConfigOutput{
			SignUpAllowed: signUpAllowedCheck,
		})
	}
}
