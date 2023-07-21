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

type ConfigsOutput struct {
	SignUpAllowed bool `json:"sign_up_allowed"`
}

// HandleListConfigs returns an http.HandlerFunc that processes an http.Request
// and returns a struct containing all system configs exposed to the users.
func HandleListConfigs(userCtrl *user.Controller, config *types.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		signUpAllowedCheck, err := userCtrl.RegisterCheck(ctx, config)
		if err != nil {
			render.TranslatedUserError(w, err)
			return
		}
		render.JSON(w, http.StatusOK, ConfigsOutput{
			SignUpAllowed: signUpAllowedCheck,
		})
	}
}
