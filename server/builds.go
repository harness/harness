package server

import (
	"io"
	"strconv"

	"github.com/drone/drone/common"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// GetBuild accepts a request to retrieve a build
// from the datastore for the given repository and
// build number.
//
//     GET /api/builds/:owner/:name/:number
//
func GetBuild(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	num, err := strconv.Atoi(c.Params.ByName("number"))
	if err != nil {
		c.Fail(400, err)
		return
	}
	build, err := store.Build(repo.FullName, num)
	if err != nil {
		c.Fail(404, err)
	} else {
		c.JSON(200, build)
	}
}

// GetBuilds accepts a request to retrieve a list
// of builds from the datastore for the given repository.
//
//     GET /api/builds/:owner/:name
//
func GetBuilds(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	builds, err := store.BuildList(repo.FullName)
	if err != nil {
		c.Fail(404, err)
	} else {
		c.JSON(200, builds)
	}
}

// GetBuildLogs accepts a request to retrieve logs from the
// datastore for the given repository, build and task
// number.
//
//     GET /api/repos/:owner/:name/logs/:number/:task
//
func GetBuildLogs(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	full, _ := strconv.ParseBool(c.Params.ByName("full"))
	build, _ := strconv.Atoi(c.Params.ByName("number"))
	task, _ := strconv.Atoi(c.Params.ByName("task"))

	r, err := store.LogReader(repo.FullName, build, task)
	if err != nil {
		c.Fail(404, err)
	} else if full {
		io.Copy(c.Writer, r)
	} else {
		io.Copy(c.Writer, io.LimitReader(r, 2000000))
	}
}

// PostBuildStatus accepts a request to create a new build
// status. The created user status is returned in JSON
// format if successful.
//
//     POST /api/repos/:owner/:name/status/:number
//
func PostBuildStatus(c *gin.Context) {
	store := ToDatastore(c)
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
	if err := store.SetStatus(repo.Name, num, in); err != nil {
		c.Fail(400, err)
	} else {
		c.JSON(201, in)
	}
}
