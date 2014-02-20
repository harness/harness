package handler

import (
	"fmt"
	"net/http"

	"github.com/drone/drone/pkg/database"
)

// Display a static badge (png format) for a specific
// repository and an optional branch.
// TODO this needs to implement basic caching
func Badge(w http.ResponseWriter, r *http.Request) error {
	branchParam := r.FormValue("branch")
	hostParam := r.FormValue(":host")
	ownerParam := r.FormValue(":owner")
	nameParam := r.FormValue(":name")
	repoSlug := fmt.Sprintf("%s/%s/%s", hostParam, ownerParam, nameParam)

	// get the repo from the database
	repo, err := database.GetRepoSlug(repoSlug)
	if err != nil {
		http.NotFound(w, r)
		return nil
	}

	// get the default branch for the repository
	// if no branch is provided.
	if len(branchParam) == 0 {
		branchParam = repo.DefaultBranch()
	}

	// default badge of "unknown"
	badge := "/img/build_unknown.png"

	// get the latest commit from the database
	// for the requested branch
	commit, err := database.GetBranch(repo.ID, branchParam)
	if err == nil {
		switch commit.Status {
		case "Success":
			badge = "/img/build_success.png"
		case "Failing", "Failure":
			badge = "/img/build_failing.png"
		}
	}

	http.Redirect(w, r, badge, http.StatusSeeOther)
	return nil
}
