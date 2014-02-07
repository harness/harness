package handler

import (
	"net/http"

	"github.com/drone/drone/pkg/database"
	. "github.com/drone/drone/pkg/model"
)

// Returns the combined stdout / stderr for an individual Build.
func BuildOut(w http.ResponseWriter, r *http.Request, u *User, repo *Repo) error {
	hash := r.FormValue(":commit")
	labl := r.FormValue(":label")

	// get the commit from the database
	commit, err := database.GetCommitHash(hash, repo.ID)
	if err != nil {
		return err
	}

	// get the build from the database
	build, err := database.GetBuildSlug(labl, commit.ID)
	if err != nil {
		return err
	}

	return RenderText(w, build.Stdout, http.StatusOK)
}

// Returns the gzipped stdout / stderr for an individual Build
func BuildOutGzip(w http.ResponseWriter, r *http.Request, u *User) error {
	// TODO
	return nil
}
