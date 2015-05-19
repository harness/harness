package server

import (
	"net"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	log "github.com/Sirupsen/logrus"
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
	err = store.SetAgent(work.Commit, addr)
	if err != nil {
		// note the we are ignoring and just logging the error here.
		// we consider this an acceptible failure because it doesn't
		// impact anything other than live-streaming output.
		log.Errorf("unable to store the agent address %s for build %s %v",
			addr, work.Repo.FullName, work.Commit.Sequence)
	}

	c.JSON(200, work)

	// acknowledge work received by the client
	queue.Ack(work)
}

// POST /queue/push/:owner/:repo
func PushCommit(c *gin.Context) {
	store := ToDatastore(c)
	repo := ToRepo(c)

	in := &common.Commit{}
	if !c.BindWith(in, binding.JSON) {
		return
	}
	user, err := store.User(repo.UserID)
	if err != nil {
		c.Fail(404, err)
		return
	}
	commit, err := store.CommitSeq(repo, in.Sequence)
	if err != nil {
		c.Fail(404, err)
		return
	}

	commit.Started = in.Started
	commit.Finished = in.Finished
	commit.State = in.State

	updater := ToUpdater(c)
	err = updater.SetCommit(user, repo, commit)
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

	in := &common.Build{}
	if !c.BindWith(in, binding.JSON) {
		return
	}

	commit, err := store.CommitSeq(repo, cnum)
	if err != nil {
		c.Fail(404, err)
		return
	}
	build, err := store.BuildSeq(commit, in.Sequence)
	if err != nil {
		c.Fail(404, err)
		return
	}

	build.Duration = in.Duration
	build.Started = in.Started
	build.Finished = in.Finished
	build.ExitCode = in.ExitCode
	build.State = in.State

	updater := ToUpdater(c)
	err = updater.SetBuild(repo, commit, build)
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

	commit, err := store.CommitSeq(repo, cnum)
	if err != nil {
		c.Fail(404, err)
		return
	}
	build, err := store.BuildSeq(commit, bnum)
	if err != nil {
		c.Fail(404, err)
		return
	}
	updater := ToUpdater(c)
	err = updater.SetLogs(repo, commit, build, c.Request.Body)
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
