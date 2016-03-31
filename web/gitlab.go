package web

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/drone/drone/router/middleware/session"
	"github.com/drone/drone/shared/token"
	"github.com/drone/drone/store"
)

func GetCommit(c *gin.Context) {
	repo := session.Repo(c)

	parsed, err := token.ParseRequest(c.Request, func(t *token.Token) (string, error) {
		return repo.Hash, nil
	})
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	if parsed.Text != repo.FullName {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	commit := c.Param("sha")
	branch := c.Query("branch")
	if len(branch) == 0 {
		branch = repo.Branch
	}

	build, err := store.GetBuildCommit(c, repo, commit, branch)
	if err != nil {
		c.AbortWithError(http.StatusNotFound, err)
		return
	}

	c.JSON(http.StatusOK, build)
}

func GetPullRequest(c *gin.Context) {
	repo := session.Repo(c)
	refs := fmt.Sprintf("refs/pull/%s/head", c.Param("number"))

	parsed, err := token.ParseRequest(c.Request, func(t *token.Token) (string, error) {
		return repo.Hash, nil
	})
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	if parsed.Text != repo.FullName {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	build, err := store.GetBuildRef(c, repo, refs)
	if err != nil {
		c.AbortWithError(http.StatusNotFound, err)
		return
	}

	c.JSON(http.StatusOK, build)
}

func RedirectSha(c *gin.Context) {
	repo := session.Repo(c)

	commit := c.Param("sha")
	branch := c.Query("branch")
	if len(branch) == 0 {
		branch = repo.Branch
	}

	build, err := store.GetBuildCommit(c, repo, commit, branch)
	if err != nil {
		c.AbortWithError(http.StatusNotFound, err)
		return
	}

	path := fmt.Sprintf("/%s/%s/%d", repo.Owner, repo.Name, build.Number)
	c.Redirect(http.StatusSeeOther, path)
}

func RedirectPullRequest(c *gin.Context) {
	repo := session.Repo(c)
	refs := fmt.Sprintf("refs/pull/%s/head", c.Param("number"))

	build, err := store.GetBuildRef(c, repo, refs)
	if err != nil {
		c.AbortWithError(http.StatusNotFound, err)
		return
	}

	path := fmt.Sprintf("/%s/%s/%d", repo.Owner, repo.Name, build.Number)
	c.Redirect(http.StatusSeeOther, path)
}
