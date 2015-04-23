package server

import (
	"io"
	"io/ioutil"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/drone/drone/common"
	"github.com/drone/drone/eventbus"
)

// TODO (bradrydzewski) the callback URL should be signed.
// TODO (bradrydzewski) we shouldn't need to fetch the Repo if specified in the URL path
// TODO (bradrydzewski) use SetRepoLast to update the last repository

// GET /queue/pull
func PollBuild(c *gin.Context) {
	queue := ToQueue(c)
	work := queue.PullAck()
	c.JSON(200, work)
}

// GET /queue/push/:owner/:repo
func PushBuild(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	bus := ToBus(c)
	in := &common.Build{}
	if !c.BindWith(in, binding.JSON) {
		return
	}
	build, err := store.Build(repo.FullName, in.Number)
	if err != nil {
		c.Fail(404, err)
		return
	}
	err = store.SetBuildState(repo.FullName, build)
	if err != nil {
		c.Fail(500, err)
		return
	}

	bus.Send(&eventbus.Event{
		Build: build,
		Repo:  repo,
	})
	if repo.Last != nil && repo.Last.Number > build.Number {
		c.Writer.WriteHeader(200)
		return
	}
	repo.Last = build
	store.SetRepo(repo)
	c.Writer.WriteHeader(200)
}

// POST /queue/push/:owner/:repo/:build
func PushTask(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	bus := ToBus(c)
	num, _ := strconv.Atoi(c.Params.ByName("build"))
	in := &common.Task{}
	if !c.BindWith(in, binding.JSON) {
		return
	}
	err := store.SetBuildTask(repo.FullName, num, in)
	if err != nil {
		c.Fail(404, err)
		return
	}
	build, err := store.Build(repo.FullName, num)
	if err != nil {
		c.Fail(404, err)
		return
	}
	bus.Send(&eventbus.Event{
		Build: build,
		Repo:  repo,
	})
	c.Writer.WriteHeader(200)
}

// POST /queue/push/:owner/:repo/:build/:task/logs
func PushLogs(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	bnum, _ := strconv.Atoi(c.Params.ByName("build"))
	tnum, _ := strconv.Atoi(c.Params.ByName("task"))

	// TODO (bradrydzewski) change this interface to accept an io.Reader
	// instead of a byte array so that we can buffer the write and so that
	// we avoid unnecessary copies of the data in memory.
	logs, err := ioutil.ReadAll(io.LimitReader(c.Request.Body, 5000000)) //5MB
	defer c.Request.Body.Close()
	if err != nil {
		c.Fail(500, err)
		return
	}
	err = store.SetLogs(repo.FullName, bnum, tnum, logs)
	if err != nil {
		c.Fail(500, err)
		return
	}
	c.Writer.WriteHeader(200)
}
