// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package projects

import (
	"net/http"

	"github.com/bradrydzewski/my-app/internal/api/render/platform"
	"github.com/bradrydzewski/my-app/types"
)

// standalone version of the product uses a single,
// hard-coded project as its default.
var defaultProject = &types.Project{
	Identifier: "default",
	Color:      "#0063f7",
	Desc:       "Default Project",
	Name:       "Default Project",
	Modules:    []string{},
	Org:        "default",
	Tags:       map[string]string{},
}

// HandleFind returns an http.HandlerFunc that writes json-encoded
// project to the http response body.
func HandleFind() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		platform.RenderResource(w, defaultProject, 200)
	}
}
