package server

import (
	"net/http"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/drone/drone/bus"
	"github.com/drone/drone/queue"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/store"
	"github.com/drone/drone/stream"
	"github.com/gin-gonic/gin"
	"github.com/square/go-jose"

	"github.com/drone/drone/model"
	"github.com/drone/drone/router/middleware/session"
)

func GetBuilds(c *gin.Context) {
	repo := session.Repo(c)
	builds, err := store.GetBuildList(c, repo)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, builds)
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

	c.JSON(http.StatusOK, &out)
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

	c.JSON(http.StatusOK, &out)
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
		// TODO implement limited streaming to avoid crashing the browser
	}

	c.Header("Content-Type", "application/json")
	stream.Copy(c.Writer, r)
}

func DeleteBuild(c *gin.Context) {
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

	if job.Status != model.StatusRunning {
		c.String(400, "Cannot cancel a non-running build")
		return
	}

	job.Status = model.StatusKilled
	job.Finished = time.Now().Unix()
	if job.Started == 0 {
		job.Started = job.Finished
	}
	job.ExitCode = 137
	store.UpdateBuildJob(c, build, job)

	bus.Publish(c, bus.NewEvent(bus.Cancelled, repo, build, job))
	c.String(204, "")
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
	config := ToConfig(c)
	raw, err := remote_.File(user, repo, build, config.Yaml)
	if err != nil {
		log.Errorf("failure to get build config for %s. %s", repo.FullName, err)
		c.AbortWithError(404, err)
		return
	}

	// Fetch secrets file but don't exit on error as it's optional
	sec, err := remote_.File(user, repo, build, config.Shasum)
	if err != nil {
		log.Debugf("cannot find build secrets for %s. %s", repo.FullName, err)
	}

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

		event := c.DefaultQuery("event", build.Event)
		if event == model.EventPush ||
			event == model.EventPull ||
			event == model.EventTag ||
			event == model.EventDeploy {
			build.Event = event
		}
		build.Deploy = c.DefaultQuery("deploy_to", build.Deploy)
	}

	// Read query string parameters into buildParams, exclude reserved params
	var buildParams = map[string]string{}
	for key, val := range c.Request.URL.Query() {
		switch key {
		case "fork", "event", "deply_to":
		default:
			// We only accept string literals, because build parameters will be
			// injected as environment variables
			buildParams[key] = val[0]
		}
	}

	// todo move this to database tier
	// and wrap inside a transaction
	build.Status = model.StatusPending
	build.Started = 0
	build.Finished = 0
	build.Enqueued = time.Now().UTC().Unix()
	for _, job := range jobs {
		for k, v := range buildParams {
			job.Environment[k] = v
		}
		job.Error = ""
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

	// get the previous build so that we can send
	// on status change notifications
	last, _ := store.GetBuildLastBefore(c, repo, build.Branch, build.ID)
	secs, err := store.GetSecretList(c, repo)
	if err != nil {
		log.Errorf("Error getting secrets for %s#%d. %s", repo.FullName, build.Number, err)
	}

	var signed bool
	var verified bool

	signature, err := jose.ParseSigned(string(sec))
	if err != nil {
		log.Debugf("cannot parse .drone.yml.sig file. %s", err)
	} else if len(sec) == 0 {
		log.Debugf("cannot parse .drone.yml.sig file. empty file")
	} else {
		signed = true
		output, err := signature.Verify([]byte(repo.Hash))
		if err != nil {
			log.Debugf("cannot verify .drone.yml.sig file. %s", err)
		} else if string(output) != string(raw) {
			log.Debugf("cannot verify .drone.yml.sig file. no match. %q <> %q", string(output), string(raw))
		} else {
			verified = true
		}
	}

	log.Debugf(".drone.yml is signed=%v and verified=%v", signed, verified)

	bus.Publish(c, bus.NewBuildEvent(bus.Enqueued, repo, build))
	for _, job := range jobs {
		queue.Publish(c, &queue.Work{
			Signed:    signed,
			Verified:  verified,
			User:      user,
			Repo:      repo,
			Build:     build,
			BuildLast: last,
			Job:       job,
			Netrc:     netrc,
			Yaml:      string(raw),
			Secrets:   secs,
			System:    &model.System{Link: httputil.GetURL(c.Request)},
		})
	}
}

func GetBuildQueue(c *gin.Context) {
	out, err := store.GetBuildQueue(c)
	if err != nil {
		c.String(500, "Error getting build queue. %s", err)
		return
	}
	c.JSON(200, out)
}
