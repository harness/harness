package server

import (
	"fmt"
	"strconv"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/gin-gonic/gin"

	"github.com/drone/drone/pkg/token"
)

// RedirectSha accepts a request to retvie a redirect
// to job from the datastore for the given repository
// and commit sha
//
//  GET /gitlab/:owner/:name/redirect/commits/:sha
//
// REASON: It required by GitLab, becuase we get only
// sha and ref name, but drone uses build numbers
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

// RedirectPullRequest accepts a request to retvie a redirect
// to job from the datastore for the given repository
// and pull request number
//
//  GET /gitlab/:owner/:name/redirect/pulls/:number
//
// REASON: It required by GitLab, because we get only
// internal merge request id/ref/sha, but drone uses
// build numbers
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

// GetPullRequest accepts a requests to retvie a pull request
// from the datastore for the given repository and
// pull request number
//
//	GET /gitlab/:owner/:name/pulls/:number
//
// REASON: It required by GitLab, becuase we get only
// sha and ref name, but drone uses build numbers
func GetPullRequest(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)

	parsed, err := token.ParseRequest(c.Request, func(t *token.Token) (string, error) {
		return repo.Hash, nil
	})
	if err != nil {
		c.Fail(400, err)
		return
	}
	if parsed.Text != repo.FullName {
		c.AbortWithStatus(403)
		return
	}

	num, err := strconv.Atoi(c.Params.ByName("number"))
	if err != nil {
		c.Fail(400, err)
		return
	}
	build, err := store.BuildPullRequestNumber(repo, num)
	if err != nil {
		c.Fail(404, err)
		return
	}
	build.Jobs, err = store.JobList(build)
	if err != nil {
		c.Fail(404, err)
	} else {
		c.JSON(200, build)
	}
}

// GetCommit accepts a requests to retvie a sha and branch
// from the datastore for the given repository and
// pull request number
//
//	GET /gitlab/:owner/:name/commits/:sha
//
// REASON: It required by GitLab, becuase we get only
// sha and ref name, but drone uses build numbers
func GetCommit(c *gin.Context) {
	var branch string

	store := ToDatastore(c)
	repo := ToRepo(c)
	sha := c.Params.ByName("sha")

	parsed, err := token.ParseRequest(c.Request, func(t *token.Token) (string, error) {
		return repo.Hash, nil
	})
	if err != nil {
		c.Fail(400, err)
		return
	}
	if parsed.Text != repo.FullName {
		c.AbortWithStatus(403)
		return
	}

	branch = c.Request.FormValue("branch")
	if branch == "" {
		branch = repo.Branch
	}

	build, err := store.BuildSha(repo, sha, branch)
	if err != nil {
		c.Fail(404, err)
		return
	}

	build.Jobs, err = store.JobList(build)
	if err != nil {
		c.Fail(404, err)
	} else {
		c.JSON(200, build)
	}

	return
}
