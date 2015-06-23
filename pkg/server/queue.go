package server

import (
	"net"
	"strconv"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/gin-gonic/gin"
	"github.com/drone/drone/Godeps/_workspace/src/github.com/gin-gonic/gin/binding"

	log "github.com/drone/drone/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	common "github.com/drone/drone/pkg/types"
)

// GET /queue/pull
func PollBuild(c *gin.Context) {
	queue := ToQueue(c)
	store := ToDatastore(c)

	// extract the IP address from the agent that is
	// polling for builds.
	host := c.Request.RemoteAddr
	addr, _, err := net.SplitHostPort(host)
	if err != nil {
		addr = host
	}
	addr = net.JoinHostPort(addr, "1999")

	log.Infof("agent connected and polling builds at %s", addr)

	// pull an item from the queue
	work := queue.PullClose(c.Writer)
	if work == nil {
		c.AbortWithStatus(500)
		return
	}

	// persist the relationship between agent and commit.
	err = store.SetAgent(work.Build, addr)
	if err != nil {
		// note the we are ignoring and just logging the error here.
		// we consider this an acceptible failure because it doesn't
		// impact anything other than live-streaming output.
		log.Errorf("unable to store the agent address %s for build %s %v",
			addr, work.Repo.FullName, work.Build.Number)
	}

	c.JSON(200, work)

	// acknowledge work received by the client
	queue.Ack(work)
}

// POST /queue/push/:owner/:repo
func PushCommit(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)

	in := &common.Build{}
	if !c.BindWith(in, binding.JSON) {
		return
	}
	user, err := store.User(repo.UserID)
	if err != nil {
		c.Fail(404, err)
		return
	}
	build, err := store.BuildNumber(repo, in.Number)
	if err != nil {
		c.Fail(404, err)
		return
	}

	build.Started = in.Started
	build.Finished = in.Finished
	build.Status = in.Status

	updater := ToUpdater(c)
	err = updater.SetBuild(user, repo, build)
	if err != nil {
		c.Fail(500, err)
		return
	}
	c.Writer.WriteHeader(200)
}

// POST /queue/push/:owner/:repo/:commit
func PushBuild(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	cnum, _ := strconv.Atoi(c.Params.ByName("commit"))

	in := &common.Job{}
	if !c.BindWith(in, binding.JSON) {
		return
	}

	build, err := store.BuildNumber(repo, cnum)
	if err != nil {
		c.Fail(404, err)
		return
	}
	job, err := store.JobNumber(build, in.Number)
	if err != nil {
		c.Fail(404, err)
		return
	}

	job.Started = in.Started
	job.Finished = in.Finished
	job.ExitCode = in.ExitCode
	job.Status = in.Status

	updater := ToUpdater(c)
	err = updater.SetJob(repo, build, job)
	if err != nil {
		c.Fail(500, err)
		return
	}
	c.Writer.WriteHeader(200)
}

// POST /queue/push/:owner/:repo/:comimt/:build/logs
func PushLogs(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)
	cnum, _ := strconv.Atoi(c.Params.ByName("commit"))
	bnum, _ := strconv.Atoi(c.Params.ByName("build"))

	build, err := store.BuildNumber(repo, cnum)
	if err != nil {
		c.Fail(404, err)
		return
	}
	job, err := store.JobNumber(build, bnum)
	if err != nil {
		c.Fail(404, err)
		return
	}
	updater := ToUpdater(c)
	err = updater.SetLogs(repo, build, job, c.Request.Body)
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
