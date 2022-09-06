// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"errors"
	"net/http"

	"github.com/harness/gitness/internal/api/guard"
	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/types/enum"
)

/*
 * Updates an existing space.
 */
func HandleUpdate(guard *guard.Guard) http.HandlerFunc {
	return guard.Space(
		enum.PermissionSpaceEdit,
		false,
		func(w http.ResponseWriter, r *http.Request) {
			/*
			 * TO-DO: Add support for updating an existing space.
			 * 		  Requires Solving:
			 *			- Update all FQNs of child spaces (or change design)
			 *			- Update all acl permissions? (or change design)
			 */
			render.BadRequest(w, errors.New("Updating an existing space is not supported."))
		})
}
