package controller

import (
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/drone/drone/engine"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/store"
	"github.com/gin-gonic/gin"

	"github.com/drone/drone/model"
	"github.com/drone/drone/router/middleware/context"
	"github.com/drone/drone/router/middleware/session"
)

func GetBuilds(c *gin.Context) {
	repo := session.Repo(c)
	builds, err := store.GetBuildList(c, repo)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.IndentedJSON(http.StatusOK, builds)
}

func GetBuild(c *gin.Context) {
	if c.Param("number") == "latest" {
		GetBuildLast(c)
		return
	}

	repo := session.Repo(c)
	num, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	build, err := store.GetBuildNumber(c, repo, num)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	jobs, _ := store.GetJobList(c, build)

	out := struct {
		*model.Build
		Jobs []*model.Job `json:"jobs"`
	}{build, jobs}

	c.IndentedJSON(http.StatusOK, &out)
}

func GetBuildLast(c *gin.Context) {
	repo := session.Repo(c)
	branch := c.DefaultQuery("branch", repo.Branch)

	build, err := store.GetBuildLast(c, repo, branch)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	jobs, _ := store.GetJobList(c, build)

	out := struct {
		*model.Build
		Jobs []*model.Job `json:"jobs"`
	}{build, jobs}

	c.IndentedJSON(http.StatusOK, &out)
}

func GetBuildLogs(c *gin.Context) {
	repo := session.Repo(c)

	// the user may specify to stream the full logs,
	// or partial logs, capped at 2MB.
	full, _ := strconv.ParseBool(c.DefaultQuery("full", "false"))

	// parse the build number and job sequence number from
	// the repquest parameter.
	num, _ := strconv.Atoi(c.Params.ByName("number"))
	seq, _ := strconv.Atoi(c.Params.ByName("job"))

	build, err := store.GetBuildNumber(c, repo, num)
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	job, err := store.GetJobNumber(c, build, seq)
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	r, err := store.ReadLog(c, job)
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	defer r.Close()
	if full {
		io.Copy(c.Writer, r)
	} else {
		io.Copy(c.Writer, io.LimitReader(r, 2000000))
	}
}

func DeleteBuild(c *gin.Context) {
	engine_ := context.Engine(c)
	repo := session.Repo(c)

	// parse the build number and job sequence number from
	// the repquest parameter.
	num, _ := strconv.Atoi(c.Params.ByName("number"))
	seq, _ := strconv.Atoi(c.Params.ByName("job"))

	build, err := store.GetBuildNumber(c, repo, num)
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	job, err := store.GetJobNumber(c, build, seq)
	if err != nil {
		c.AbortWithError(404, err)
		return
	}
	node, err := store.GetNode(c, job.NodeID)
	if err != nil {
		c.AbortWithError(404, err)
		return
	}
	engine_.Cancel(build.ID, job.ID, node)
}

func PostBuild(c *gin.Context) {

	remote_ := remote.FromContext(c)
	repo := session.Repo(c)
	fork := c.DefaultQuery("fork", "false")

	num, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	user, err := store.GetUser(c, repo.UserID)
	if err != nil {
		log.Errorf("failure to find repo owner %s. %s", repo.FullName, err)
		c.AbortWithError(500, err)
		return
	}

	build, err := store.GetBuildNumber(c, repo, num)
	if err != nil {
		log.Errorf("failure to get build %d. %s", num, err)
		c.AbortWithError(404, err)
		return
	}

	// if the remote has a refresh token, the current access token
	// may be stale. Therefore, we should refresh prior to dispatching
	// the job.
	if refresher, ok := remote_.(remote.Refresher); ok {
		ok, _ := refresher.Refresh(user)
		if ok {
			store.UpdateUser(c, user)
		}
	}

	// fetch the .drone.yml file from the database
	raw, sec, err := remote_.Script(user, repo, build)
	if err != nil {
		log.Errorf("failure to get .drone.yml for %s. %s", repo.FullName, err)
		c.AbortWithError(404, err)
		return
	}

	key, _ := store.GetKey(c, repo)
	netrc, err := remote_.Netrc(user, repo)
	if err != nil {
		log.Errorf("failure to generate netrc for %s. %s", repo.FullName, err)
		c.AbortWithError(500, err)
		return
	}

	jobs, err := store.GetJobList(c, build)
	if err != nil {
		log.Errorf("failure to get build %d jobs. %s", build.Number, err)
		c.AbortWithError(404, err)
		return
	}

	// must not restart a running build
	if build.Status == model.StatusPending || build.Status == model.StatusRunning {
		c.String(409, "Cannot re-start a started build")
		return
	}

	// forking the build creates a duplicate of the build
	// and then executes. This retains prior build history.
	if forkit, _ := strconv.ParseBool(fork); forkit {
		build.ID = 0
		build.Number = 0
		for _, job := range jobs {
			job.ID = 0
			job.NodeID = 0
		}
		err := store.CreateBuild(c, build, jobs...)
		if err != nil {
			c.String(500, err.Error())
			return
		}
	}

	// todo move this to database tier
	// and wrap inside a transaction
	build.Status = model.StatusPending
	build.Started = 0
	build.Finished = 0
	build.Enqueued = time.Now().UTC().Unix()
	for _, job := range jobs {
		job.Status = model.StatusPending
		job.Started = 0
		job.Finished = 0
		job.ExitCode = 0
		job.NodeID = 0
		job.Enqueued = build.Enqueued
		store.UpdateJob(c, job)
	}

	err = store.UpdateBuild(c, build)
	if err != nil {
		c.AbortWithStatus(500)
		return
	}

	c.JSON(202, build)

	// get the previous build so taht we can send
	// on status change notifications
	last, _ := store.GetBuildLastBefore(c, repo, build.Branch, build.ID)

	engine_ := context.Engine(c)
	go engine_.Schedule(c.Copy(), &engine.Task{
		User:      user,
		Repo:      repo,
		Build:     build,
		BuildPrev: last,
		Jobs:      jobs,
		Keys:      key,
		Netrc:     netrc,
		Config:    string(raw),
		Secret:    string(sec),
		System: &model.System{
			Link:      httputil.GetURL(c.Request),
			Plugins:   strings.Split(os.Getenv("PLUGIN_FILTER"), " "),
			Globals:   strings.Split(os.Getenv("PLUGIN_PARAMS"), " "),
			Escalates: strings.Split(os.Getenv("ESCALATE_FILTER"), " "),
		},
	})

}
