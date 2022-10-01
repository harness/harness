// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package serviceaccount

import (
	"net/http"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/internal/api/request"
	"github.com/harness/gitness/internal/guard"
	"github.com/harness/gitness/types/enum"
)

// HandleFind returns an http.HandlerFunc that writes json-encoded
// service account information to the http response body.
func HandleFind(guard *guard.Guard) http.HandlerFunc {
	return guard.ServiceAccount(
		enum.PermissionServiceAccountView,
		func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			sa, _ := request.ServiceAccountFrom(ctx)
			render.JSON(w, http.StatusOK, sa)
		})
}
