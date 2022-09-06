// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"errors"
	"net/http"

	"github.com/harness/gitness/internal/api/guard"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/types/enum"
)

/*
 * Updates an existing repository.
 */
func HandleUpdate(guard *guard.Guard) http.HandlerFunc {
	return guard.Repo(
		enum.PermissionRepoEdit,
		false,
		func(w http.ResponseWriter, r *http.Request) {
			/*
			 * TO-DO: Add support for updating an existing repository.
			 */
			render.BadRequest(w, errors.New("Updating an existing repo is not supported."))
		})
}
