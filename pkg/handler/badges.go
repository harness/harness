package handler

import (
	"fmt"
	"net/http"

	"github.com/drone/drone/pkg/database"
)

const (
	badgeSuccess = "https://img.shields.io/badge/build-success-brightgreen.svg"
	badgeFailure = "https://img.shields.io/badge/build-failure-red.svg"
	badgeUnknown = "https://img.shields.io/badge/build-unknown-lightgray.svg"
)

// Display a static badge (svg format) for a specific
// repository and an optional branch.
// TODO this needs to implement basic caching
func Badge(w http.ResponseWriter, r *http.Request) error {
	successParam := r.FormValue("success")
	failureParam := r.FormValue("failure")
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

	var badge string

	// get the latest commit from the database
	// for the requested branch
	commit, err := database.GetBranch(repo.ID, branchParam)
	if err != nil {
		http.NotFound(w, r)
		return nil
	}

	switch {
	case commit.Status == "Success" && len(successParam) == 0:
		// if no success image is provided, we serve a
		// badge using the shields.io service
		badge = badgeSuccess
	case commit.Status == "Success" && len(successParam) != 0:
		// otherwise we serve the user defined success badge
		badge = successParam
	case commit.Status == "Failure" && len(failureParam) == 0:
		// if no failure image is provided, we serve a
		// badge using the shields.io service
		badge = badgeFailure
	case commit.Status == "Failure" && len(failureParam) != 0:
		// otherwise we serve the user defined failure badge
		badge = failureParam
	default:
		// otherwise load unknown image
		badge = badgeUnknown
	}

	http.Redirect(w, r, badge, http.StatusSeeOther)
	return nil
}
