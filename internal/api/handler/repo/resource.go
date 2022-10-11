package repo

import (
	"net/http"
	"strings"

	"github.com/harness/gitness/internal/api/render"
	"github.com/harness/gitness/resources"
)

func HandleGitIgnore() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entries, err := resources.Gitignore.ReadDir("gitignore")
		files := make([]string, len(entries))
		if err != nil {
			render.ErrorMessagef(w, http.StatusInternalServerError, "error loading gitignore files: %v", err)
			return
		}
		for i, filename := range entries {
			files[i] = strings.ReplaceAll(filename.Name(), ".gitignore", "")
		}
		render.JSON(w, http.StatusOK, files)
	}
}

func HandleLicence() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response, err := resources.Licence.ReadFile("licence/index.json")
		if err != nil {
			render.ErrorMessagef(w, http.StatusInternalServerError, "error loading licence file: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(response)
	}
}
