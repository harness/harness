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
	"github.com/gin-gonic/gin"

	"github.com/drone/drone/model"
	"github.com/drone/drone/router/middleware/context"
	"github.com/drone/drone/router/middleware/session"
)

func GetBuilds(c *gin.Context) {
	repo := session.Repo(c)
	db := context.Database(c)
	builds, err := model.GetBuildList(db, repo)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.IndentedJSON(http.StatusOK, builds)
}

func GetBuild(c *gin.Context) {
	repo := session.Repo(c)
	db := context.Database(c)

	num, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	build, err := model.GetBuildNumber(db, repo, num)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	jobs, _ := model.GetJobList(db, build)

	out := struct {
		*model.Build
		Jobs []*model.Job `json:"jobs"`
	}{build, jobs}

	c.IndentedJSON(http.StatusOK, &out)
}

func GetBuildLogs(c *gin.Context) {
	repo := session.Repo(c)
	db := context.Database(c)

	// the user may specify to stream the full logs,
	// or partial logs, capped at 2MB.
	full, _ := strconv.ParseBool(c.Params.ByName("full"))

	// parse the build number and job sequence number from
	// the repquest parameter.
	num, _ := strconv.Atoi(c.Params.ByName("number"))
	seq, _ := strconv.Atoi(c.Params.ByName("job"))

	build, err := model.GetBuildNumber(db, repo, num)
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	job, err := model.GetJobNumber(db, build, seq)
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	r, err := model.GetLog(db, job)
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
	db := context.Database(c)

	// parse the build number and job sequence number from
	// the repquest parameter.
	num, _ := strconv.Atoi(c.Params.ByName("number"))
	seq, _ := strconv.Atoi(c.Params.ByName("job"))

	build, err := model.GetBuildNumber(db, repo, num)
	if err != nil {
		c.AbortWithError(404, err)
		return
	}

	job, err := model.GetJobNumber(db, build, seq)
	if err != nil {
		c.AbortWithError(404, err)
		return
	}
	node, err := model.GetNode(db, job.NodeID)
	if err != nil {
		c.AbortWithError(404, err)
		return
	}
	engine_.Cancel(build.ID, job.ID, node)
}

func PostBuild(c *gin.Context) {

	remote_ := context.Remote(c)
	repo := session.Repo(c)
	db := context.Database(c)

	num, err := strconv.Atoi(c.Param("number"))
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	user, err := model.GetUser(db, repo.UserID)
	if err != nil {
		log.Errorf("failure to find repo owner %s. %s", repo.FullName, err)
		c.AbortWithError(500, err)
		return
	}

	build, err := model.GetBuildNumber(db, repo, num)
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
			model.UpdateUser(db, user)
		}
	}

	// fetch the .drone.yml file from the database
	raw, sec, err := remote_.Script(user, repo, build)
	if err != nil {
		log.Errorf("failure to get .drone.yml for %s. %s", repo.FullName, err)
		c.AbortWithError(404, err)
		return
	}

	key, _ := model.GetKey(db, repo)
	netrc, err := remote_.Netrc(user, repo)
	if err != nil {
		log.Errorf("failure to generate netrc for %s. %s", repo.FullName, err)
		c.AbortWithError(500, err)
		return
	}

	jobs, err := model.GetJobList(db, build)
	if err != nil {
		log.Errorf("failure to get build %d jobs. %s", build.Number, err)
		c.AbortWithError(404, err)
		return
	}

	// must not restart a running build
	if build.Status == model.StatusPending || build.Status == model.StatusRunning {
		c.AbortWithStatus(409)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		c.AbortWithStatus(500)
		return
	}
	defer tx.Rollback()

	build.Status = model.StatusPending
	build.Started = 0
	build.Finished = 0
	build.Enqueued = time.Now().UTC().Unix()
	for _, job := range jobs {
		job.Status = model.StatusPending
		job.Started = 0
		job.Finished = 0
		job.ExitCode = 0
		job.Enqueued = build.Enqueued
		model.UpdateJob(db, job)
	}

	err = model.UpdateBuild(db, build)
	if err != nil {
		c.AbortWithStatus(500)
		return
	}

	tx.Commit()

	c.JSON(202, build)

	engine_ := context.Engine(c)
	go engine_.Schedule(&engine.Task{
		User:   user,
		Repo:   repo,
		Build:  build,
		Jobs:   jobs,
		Keys:   key,
		Netrc:  netrc,
		Config: string(raw),
		Secret: string(sec),
		System: &model.System{
			Link:    httputil.GetURL(c.Request),
			Plugins: strings.Split(os.Getenv("PLUGIN_FILTER"), " "),
			Globals: strings.Split(os.Getenv("PLUGIN_PARAMS"), " "),
		},
	})

}
