package server

import (
	"fmt"
	"strconv"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/gin-gonic/gin"
)

// RedirectSha accepts a request to retvie a redirect
// to job from the datastore for the given repository
// and commit sha
//
//  GET /redirect/:owner/:name/commit/:sha
//
func RedirectSha(c *gin.Context) {
	var branch string

	store := ToDatastore(c)
	repo := ToRepo(c)
	sha := c.Params.ByName("sha")

	branch = c.Request.FormValue("branch")
	if branch == "" {
		branch = repo.Branch
	}

	build, err := store.BuildSha(repo, sha, branch)
	if err != nil {
		c.Redirect(301, "/")
		return
	}

	c.Redirect(301, fmt.Sprintf("/%s/%s/%d", repo.Owner, repo.Name, build.ID))
	return
}

// RedirectSha accepts a request to retvie a redirect
// to job from the datastore for the given repository
// and pull request number
//
//  GET /redirect/:owner/:name/pr/:number
//
func RedirectPullRequest(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	num, err := strconv.Atoi(c.Params.ByName("number"))
	if err != nil {
		c.Redirect(301, "/")
		return
	}

	build, err := store.BuildPullRequestNumber(repo, num)
	if err != nil {
		c.Redirect(301, "/")
		return
	}

	c.Redirect(301, fmt.Sprintf("/%s/%s/%d", repo.Owner, repo.Name, build.ID))
	return
}
