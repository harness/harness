package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/cncd/pipeline/pipeline/rpc"
	"github.com/cncd/pubsub"
	"github.com/cncd/queue"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/store"
	"github.com/gin-gonic/gin"

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
	copyLogs(c.Writer, r)
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

	config.queue.Error(context.Background(), fmt.Sprint(job.ID), queue.ErrCancel)
	c.String(204, "")
}

func PostApproval(c *gin.Context) {
	var (
		remote_ = remote.FromContext(c)
		repo    = session.Repo(c)
		user    = session.User(c)
		num, _  = strconv.Atoi(
			c.Params.ByName("number"),
		)
	)

	build, err := store.GetBuildNumber(c, repo, num)
	if err != nil {
		c.AbortWithError(404, err)
		return
	}
	if build.Status != model.StatusBlocked {
		c.String(500, "cannot decline a build with status %s", build.Status)
		return
	}
	build.Status = model.StatusPending
	build.Reviewed = time.Now().Unix()
	build.Reviewer = user.Login

	//
	//
	// This code is copied pasted until I have a chance
	// to refactor into a proper function. Lots of changes
	// and technical debt. No judgement please!
	//
	//

	// fetch the build file from the database
	raw, err := remote_.File(user, repo, build, repo.Config)
	if err != nil {
		logrus.Errorf("failure to get build config for %s. %s", repo.FullName, err)
		c.AbortWithError(404, err)
		return
	}

	netrc, err := remote_.Netrc(user, repo)
	if err != nil {
		c.String(500, "Failed to generate netrc file. %s", err)
		return
	}

	if uerr := store.UpdateBuild(c, build); err != nil {
		c.String(500, "error updating build. %s", uerr)
		return
	}

	c.JSON(200, build)

	// get the previous build so that we can send
	// on status change notifications
	last, _ := store.GetBuildLastBefore(c, repo, build.Branch, build.ID)
	secs, err := store.GetMergedSecretList(c, repo)
	if err != nil {
		logrus.Debugf("Error getting secrets for %s#%d. %s", repo.FullName, build.Number, err)
	}

	defer func() {
		uri := fmt.Sprintf("%s/%s/%d", httputil.GetURL(c.Request), repo.FullName, build.Number)
		err = remote_.Status(user, repo, build, uri)
		if err != nil {
			logrus.Errorf("error setting commit status for %s/%d", repo.FullName, build.Number)
		}
	}()

	b := builder{
		Repo:  repo,
		Curr:  build,
		Last:  last,
		Netrc: netrc,
		Secs:  secs,
		Link:  httputil.GetURL(c.Request),
		Yaml:  string(raw),
	}
	items, err := b.Build()
	if err != nil {
		build.Status = model.StatusError
		build.Started = time.Now().Unix()
		build.Finished = build.Started
		build.Error = err.Error()
		store.UpdateBuild(c, build)
		return
	}

	for _, item := range items {
		build.Jobs = append(build.Jobs, item.Job)
		store.CreateJob(c, item.Job)
		// TODO err
	}

	//
	// publish topic
	//
	message := pubsub.Message{
		Labels: map[string]string{
			"repo":    repo.FullName,
			"private": strconv.FormatBool(repo.IsPrivate),
		},
	}
	message.Data, _ = json.Marshal(model.Event{
		Type:  model.Enqueued,
		Repo:  *repo,
		Build: *build,
	})
	// TODO remove global reference
	config.pubsub.Publish(c, "topic/events", message)
	//
	// end publish topic
	//

	for _, item := range items {
		task := new(queue.Task)
		task.ID = fmt.Sprint(item.Job.ID)
		task.Labels = map[string]string{}
		task.Labels["platform"] = item.Platform
		for k, v := range item.Labels {
			task.Labels[k] = v
		}

		task.Data, _ = json.Marshal(rpc.Pipeline{
			ID:      fmt.Sprint(item.Job.ID),
			Config:  item.Config,
			Timeout: b.Repo.Timeout,
		})

		config.logger.Open(context.Background(), task.ID)
		config.queue.Push(context.Background(), task)
	}
}

func PostDecline(c *gin.Context) {
	var (
		remote_ = remote.FromContext(c)
		repo    = session.Repo(c)
		user    = session.User(c)
		num, _  = strconv.Atoi(
			c.Params.ByName("number"),
		)
	)

	build, err := store.GetBuildNumber(c, repo, num)
	if err != nil {
		c.AbortWithError(404, err)
		return
	}
	if build.Status != model.StatusBlocked {
		c.String(500, "cannot decline a build with status %s", build.Status)
		return
	}
	build.Status = model.StatusDeclined
	build.Reviewed = time.Now().Unix()
	build.Reviewer = user.Login

	err = store.UpdateBuild(c, build)
	if err != nil {
		c.String(500, "error updating build. %s", err)
		return
	}

	uri := fmt.Sprintf("%s/%s/%d", httputil.GetURL(c.Request), repo.FullName, build.Number)
	err = remote_.Status(user, repo, build, uri)
	if err != nil {
		logrus.Errorf("error setting commit status for %s/%d", repo.FullName, build.Number)
	}

	c.JSON(200, build)
}

func GetBuildQueue(c *gin.Context) {
	out, err := store.GetBuildQueue(c)
	if err != nil {
		c.String(500, "Error getting build queue. %s", err)
		return
	}
	c.JSON(200, out)
}

// copyLogs copies the stream from the source to the destination in valid JSON
// format. This converts the logs, which are per-line JSON objects, to a
// proper JSON array.
func copyLogs(dest io.Writer, src io.Reader) error {
	io.WriteString(dest, "[")

	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		io.WriteString(dest, scanner.Text())
		io.WriteString(dest, ",\n")
	}

	io.WriteString(dest, "{}]")

	return nil
}

//
//
//
//
//
//

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
		logrus.Errorf("failure to find repo owner %s. %s", repo.FullName, err)
		c.AbortWithError(500, err)
		return
	}

	build, err := store.GetBuildNumber(c, repo, num)
	if err != nil {
		logrus.Errorf("failure to get build %d. %s", num, err)
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
	raw, err := remote_.File(user, repo, build, repo.Config)
	if err != nil {
		logrus.Errorf("failure to get build config for %s. %s", repo.FullName, err)
		c.AbortWithError(404, err)
		return
	}

	netrc, err := remote_.Netrc(user, repo)
	if err != nil {
		logrus.Errorf("failure to generate netrc for %s. %s", repo.FullName, err)
		c.AbortWithError(500, err)
		return
	}

	jobs, err := store.GetJobList(c, build)
	if err != nil {
		logrus.Errorf("failure to get build %d jobs. %s", build.Number, err)
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
		build.Parent = num
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
		case "fork", "event", "deploy_to":
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
	build.Error = ""
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
	secs, err := store.GetMergedSecretList(c, repo)
	if err != nil {
		logrus.Debugf("Error getting secrets for %s#%d. %s", repo.FullName, build.Number, err)
	}

	b := builder{
		Repo:  repo,
		Curr:  build,
		Last:  last,
		Netrc: netrc,
		Secs:  secs,
		Link:  httputil.GetURL(c.Request),
		Yaml:  string(raw),
	}
	items, err := b.Build()
	if err != nil {
		build.Status = model.StatusError
		build.Started = time.Now().Unix()
		build.Finished = build.Started
		build.Error = err.Error()
		return
	}

	for i, item := range items {
		// TODO prevent possible index out of bounds
		item.Job.ID = jobs[i].ID
		build.Jobs = append(build.Jobs, item.Job)
		store.UpdateJob(c, item.Job)
	}

	//
	// publish topic
	//
	message := pubsub.Message{
		Labels: map[string]string{
			"repo":    repo.FullName,
			"private": strconv.FormatBool(repo.IsPrivate),
		},
	}
	message.Data, _ = json.Marshal(model.Event{
		Type:  model.Enqueued,
		Repo:  *repo,
		Build: *build,
	})
	// TODO remove global reference
	config.pubsub.Publish(c, "topic/events", message)
	//
	// end publish topic
	//

	for _, item := range items {
		task := new(queue.Task)
		task.ID = fmt.Sprint(item.Job.ID)
		task.Labels = map[string]string{}
		task.Labels["platform"] = item.Platform
		for k, v := range item.Labels {
			task.Labels[k] = v
		}

		task.Data, _ = json.Marshal(rpc.Pipeline{
			ID:      fmt.Sprint(item.Job.ID),
			Config:  item.Config,
			Timeout: b.Repo.Timeout,
		})

		config.logger.Open(context.Background(), task.ID)
		config.queue.Push(context.Background(), task)
	}
}
