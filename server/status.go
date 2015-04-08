package server

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/drone/drone/common"
)

// GetStatus accepts a request to retrieve a build status
// from the datastore for the given repository and
// build number.
//
//     GET /api/status/:owner/:name/:number/:context
//
func GetStatus(c *gin.Context) {
	ds := ToDatastore(c)
	repo := ToRepo(c)
	num, _ := strconv.Atoi(c.Params.ByName("number"))
	ctx := c.Params.ByName("context")

	status, err := ds.GetBuildStatus(repo.FullName, num, ctx)
	if err != nil {
		c.Fail(404, err)
	} else {
		c.JSON(200, status)
	}
}

// PostStatus accepts a request to create a new build
// status. The created user status is returned in JSON
// format if successful.
//
//     POST /api/status/:owner/:name/:number
//
func PostStatus(c *gin.Context) {
	ds := ToDatastore(c)
	repo := ToRepo(c)
	num, err := strconv.Atoi(c.Params.ByName("number"))
	if err != nil {
		c.Fail(400, err)
		return
	}
	in := &common.Status{}
	if !c.BindWith(in, binding.JSON) {
		c.AbortWithStatus(400)
		return
	}
	if err := ds.InsertBuildStatus(repo.Name, num, in); err != nil {
		c.Fail(400, err)
	} else {
		c.JSON(201, in)
	}
}

// GetStatusList accepts a request to retrieve a list of
// all build status from the datastore for the given repository
// and build number.
//
//     GET /api/status/:owner/:name/:number/:context
//
func GetStatusList(c *gin.Context) {
	ds := ToDatastore(c)
	repo := ToRepo(c)
	num, _ := strconv.Atoi(c.Params.ByName("number"))

	list, err := ds.GetBuildStatusList(repo.FullName, num)
	if err != nil {
		c.Fail(404, err)
	} else {
		c.JSON(200, list)
	}
}
