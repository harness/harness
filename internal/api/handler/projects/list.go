// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package projects

import (
	"net/http"

	"github.com/harness/scm/internal/api/render/platform"
	"github.com/harness/scm/types"
)

// standalone version of the product uses a single,
// hard-coded project as its default.
var defaultProjectList = &types.ProjectList{
	Data:          []*types.Project{defaultProject},
	Empty:         false,
	PageIndex:     1,
	PageItemCount: 1,
	PageSize:      1,
	TotalItems:    1,
	TotalPages:    1,
}

// HandleList returns an http.HandlerFunc that writes json-encoded
// project list to the http response body.
func HandleList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		platform.RenderResource(w, defaultProjectList, 200)
	}
}
