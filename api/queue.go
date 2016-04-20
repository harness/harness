package api

import (
	"fmt"
	"io"
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/drone/drone/bus"
	"github.com/drone/drone/model"
	"github.com/drone/drone/queue"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/store"
	"github.com/drone/drone/stream"
	"github.com/gin-gonic/gin"
)

// Pull is a long request that polls and attemts to pull work off the queue stack.
func Pull(c *gin.Context) {
	logrus.Debugf("Agent %s connected.", c.ClientIP())

	w := queue.PullClose(c, c.Writer)
	if w == nil {
		logrus.Debugf("Agent %s could not pull work.", c.ClientIP())
	} else {
		c.JSON(202, w)

		logrus.Debugf("Agent %s assigned work. %s/%s#%d.%d",
			c.ClientIP(),
			w.Repo.Owner,
			w.Repo.Name,
			w.Build.Number,
			w.Job.Number,
		)
	}
}

// Wait is a long request that polls and waits for cancelled build requests.
func Wait(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(500, "Invalid input. %s", err)
		return
	}

	eventc := make(chan *bus.Event, 1)

	bus.Subscribe(c, eventc)
	defer bus.Unsubscribe(c, eventc)

	for {
		select {
		case event := <-eventc:
			if event.Job.ID == id &&
				event.Job.Status != model.StatusPending &&
				event.Job.Status != model.StatusRunning {
				c.JSON(200, event.Job)
				return
			}
		case <-c.Writer.CloseNotify():
			return
		}
	}
}

// Update handles build updates from the agent  and persists to the database.
func Update(c *gin.Context) {
	work := &queue.Work{}
	if err := c.BindJSON(work); err != nil {
		logrus.Errorf("Invalid input. %s", err)
		return
	}

	// TODO(bradrydzewski) it is really annoying that we have to do this lookup
	// and I'd prefer not to. The reason we do this is because the Build and Job
	// have fields that aren't serialized to json and would be reset to their
	// empty values if we just saved what was coming in the http.Request body.
	build, err := store.GetBuild(c, work.Build.ID)
	if err != nil {
		c.String(404, "Unable to find build. %s", err)
		return
	}
	job, err := store.GetJob(c, work.Job.ID)
	if err != nil {
		c.String(404, "Unable to find job. %s", err)
		return
	}
	build.Started = work.Build.Started
	build.Finished = work.Build.Finished
	build.Status = work.Build.Status
	job.Started = work.Job.Started
	job.Finished = work.Job.Finished
	job.Status = work.Job.Status
	job.ExitCode = work.Job.ExitCode

	ok, err := store.UpdateBuildJob(c, build, job)
	if err != nil {
		c.String(500, "Unable to update job. %s", err)
		return
	}

	if ok {
		// get the user because we transfer the user form the server to agent
		// and back we lose the token which does not get serialized to json.
		user, err := store.GetUser(c, work.User.ID)
		if err != nil {
			c.String(500, "Unable to find user. %s", err)
			return
		}
		bus.Publish(c, &bus.Event{})
		remote.Status(c, user, work.Repo, build,
			fmt.Sprintf("%s/%s/%d", work.System.Link, work.Repo.FullName, work.Build.Number))
	}
	c.JSON(200, work)
}

// Stream streams the logs to disk or memory for broadcasing to listeners. Once
// the stream is closed it is moved to permanent storage in the database.
func Stream(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(500, "Invalid input. %s", err)
		return
	}

	key := stream.ToKey(id)
	rc, wc, err := stream.Create(c, key)
	if err != nil {
		logrus.Errorf("Agent %s failed to create stream. %s.", c.ClientIP(), err)
		return
	}

	defer func() {
		wc.Close()
		rc.Close()
		stream.Remove(c, key)
	}()

	io.Copy(wc, c.Request.Body)
	wc.Close()

	rcc, _, err := stream.Open(c, key)
	if err != nil {
		logrus.Errorf("Agent %s failed to read cache. %s.", c.ClientIP(), err)
		return
	}
	defer func() {
		rcc.Close()
	}()

	store.WriteLog(c, &model.Job{ID: id}, rcc)
	c.String(200, "")

	logrus.Debugf("Agent %s wrote stream to database", c.ClientIP())
}
