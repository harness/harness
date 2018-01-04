package server

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/gin-gonic/gin"

	"github.com/drone/drone/model"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/store"
)

var (
	badgeSuccessUrl = `https://img.shields.io/badge/drone.io-success-brightgreen.svg`
	badgeFailureUrl = `https://img.shields.io/badge/drone.io-failure-red.svg`
	badgeStartedUrl = `https://img.shields.io/badge/drone.io-started-yellow.svg`
	badgeErrorUrl   = `https://img.shields.io/badge/drone.io-error-lightgrey.svg`
	badgeNoneUrl    = `https://img.shields.io/badge/drone.io-none-lightgrey.svg`
)

func GetBadge(c *gin.Context) {
	repo, err := store.GetRepoOwnerName(c,
		c.Param("owner"),
		c.Param("name"),
	)
	if err != nil {
		c.AbortWithStatus(404)
		return
	}

	// if no commit was found then display
	// the 'none' badge, instead of throwing
	// an error response
	branch := c.Query("branch")
	if len(branch) == 0 {
		branch = repo.Branch
	}

	queryString := c.Request.URL.RawQuery
	build, err := store.GetBuildLast(c, repo, branch)
	if err != nil {
		log.Warning(err)
		c.Redirect(302, badgeNoneUrl+"?"+queryString)
		return
	}

	switch build.Status {
	case model.StatusSuccess:
		c.Redirect(302, badgeSuccessUrl+"?"+queryString)
	case model.StatusFailure:
		c.Redirect(302, badgeFailureUrl+"?"+queryString)
	case model.StatusError, model.StatusKilled:
		c.Redirect(302, badgeErrorUrl+"?"+queryString)
	case model.StatusPending, model.StatusRunning:
		c.Redirect(302, badgeStartedUrl+"?"+queryString)
	default:
		c.Redirect(302, badgeNoneUrl+"?"+queryString)
	}
}

func GetCC(c *gin.Context) {
	repo, err := store.GetRepoOwnerName(c,
		c.Param("owner"),
		c.Param("name"),
	)
	if err != nil {
		c.AbortWithStatus(404)
		return
	}

	builds, err := store.GetBuildList(c, repo)
	if err != nil || len(builds) == 0 {
		c.AbortWithStatus(404)
		return
	}

	url := fmt.Sprintf("%s/%s/%d", httputil.GetURL(c.Request), repo.FullName, builds[0].Number)
	cc := model.NewCC(repo, builds[0], url)
	c.XML(200, cc)
}
