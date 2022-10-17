// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"net/http"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/resources"
)

func HandleGitIgnore() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		files, err := resources.GitIgnores()
		if err != nil {
			render.ErrorMessagef(w, http.StatusInternalServerError, "error loading gitignore files: %v", err)
			return
		}
		render.JSON(w, http.StatusOK, files)
	}
}

func HandleLicence() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response, err := resources.Licenses()
		if err != nil {
			render.ErrorMessagef(w, http.StatusInternalServerError, "error loading licence file: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(response)
	}
}
