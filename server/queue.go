package server

import (
	"io"
	"io/ioutil"
	"net"
	"strconv"

	log "github.com/Sirupsen/logrus"
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
	store := ToDatastore(c)
	agent := &common.Agent{
		Addr: c.Request.RemoteAddr,
	}

	// extact the host port and name and
	// replace with the default agent port (1999)
	host, _, err := net.SplitHostPort(agent.Addr)
	if err == nil {
		agent.Addr = host
	}
	agent.Addr = net.JoinHostPort(agent.Addr, "1999")

	log.Infof("agent connected and polling builds at %s", agent.Addr)

	work := queue.PullClose(c.Writer)
	if work == nil {
		c.AbortWithStatus(500)
		return
	}

	// TODO (bradrydzewski) decide how we want to handle a failure here
	// still not sure exact behavior we want ...
	err = store.SetBuildAgent(work.Repo.FullName, work.Build.Number, agent)
	if err != nil {
		log.Errorf("error persisting build agent. %s", err)
	}

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

	if in.State != common.StatePending && in.State != common.StateRunning {
		store.DelBuildAgent(repo.FullName, build.Number)
	}

	build.Duration = in.Duration
	build.Started = in.Started
	build.Finished = in.Finished
	build.State = in.State
	err = store.SetBuildState(repo.FullName, build)
	if err != nil {
		c.Fail(500, err)
		return
	}

	if build.State != common.StatePending && build.State != common.StateRunning {
		if repo.Last == nil || build.Number >= repo.Last.Number {
			repo.Last = build
			store.SetRepo(repo)
		}
	}

	// <-- FIXME
	// for some reason the Repo and Build fail to marshal to JSON.
	// It has something to do with memory / pointers. So it goes away
	// if I just refetch these items. Needs to be fixed in the future,
	// but for now should be ok
	repo, err = store.Repo(repo.FullName)
	if err != nil {
		c.Fail(500, err)
		return
	}
	build, err = store.Build(repo.FullName, in.Number)
	if err != nil {
		c.Fail(404, err)
		return
	}
	// END FIXME -->

	bus.Send(&eventbus.Event{
		Build: build,
		Repo:  repo,
	})

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
		Task:  in,
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

func GetQueue(c *gin.Context) {
	queue := ToQueue(c)
	items := queue.Items()
	c.JSON(200, items)
}
