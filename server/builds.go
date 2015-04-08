package server

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetBuild accepts a request to retrieve a build
// from the datastore for the given repository and
// build number.
//
//     GET /api/builds/:owner/:name/:number
//
func GetBuild(c *gin.Context) {
	ds := ToDatastore(c)
	repo := ToRepo(c)
	num, err := strconv.Atoi(c.Params.ByName("number"))
	if err != nil {
		c.Fail(400, err)
		return
	}
	build, err := ds.GetBuild(repo.FullName, num)
	if err != nil {
		c.Fail(404, err)
	} else {
		c.JSON(200, build)
	}
}

// GetBuild accepts a request to retrieve a list
// of builds from the datastore for the given repository.
//
//     GET /api/builds/:owner/:name
//
func GetBuilds(c *gin.Context) {
	ds := ToDatastore(c)
	repo := ToRepo(c)
	builds, err := ds.GetBuildList(repo.FullName)
	if err != nil {
		c.Fail(404, err)
	} else {
		c.JSON(200, builds)
	}
}
