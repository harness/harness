package server

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/drone/drone/bus"
	"github.com/drone/drone/model"
	"github.com/drone/drone/queue"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/store"
	"github.com/drone/drone/stream"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Pull is a long request that polls and attemts to pull work off the queue stack.
func Pull(c *gin.Context) {
	logrus.Debugf("Agent %s connected.", c.ClientIP())

	w := queue.PullClose(c, c.Writer)
	if w == nil {
		logrus.Debugf("Agent %s could not pull work.", c.ClientIP())
	} else {

		// setup the channel to stream logs
		if err := stream.Create(c, stream.ToKey(w.Job.ID)); err != nil {
			logrus.Errorf("Unable to create stream. %s", err)
		}

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
			if event.Job.ID == id && event.Type == bus.Cancelled {
				c.JSON(200, event.Job)
				return
			}
		case <-c.Writer.CloseNotify():
			return
		}
	}
}

// Update handles build updates from the agent and persists to the database.
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
	job.Error = work.Job.Error

	if build.Status == model.StatusPending {
		build.Started = work.Job.Started
		build.Status = model.StatusRunning
		store.UpdateBuild(c, build)
	}

	// if job.Status == model.StatusRunning {
	// 	err := stream.Create(c, stream.ToKey(job.ID))
	// 	if err != nil {
	// 		logrus.Errorf("Unable to create stream. %s", err)
	// 	}
	// }

	ok, err := store.UpdateBuildJob(c, build, job)
	if err != nil {
		c.String(500, "Unable to update job. %s", err)
		return
	}

	if ok && build.Status != model.StatusRunning {
		// get the user because we transfer the user form the server to agent
		// and back we lose the token which does not get serialized to json.
		user, err := store.GetUser(c, work.User.ID)
		if err != nil {
			c.String(500, "Unable to find user. %s", err)
			return
		}
		remote.Status(c, user, work.Repo, build,
			fmt.Sprintf("%s/%s/%d", work.System.Link, work.Repo.FullName, work.Build.Number))
	}

	if build.Status == model.StatusRunning {
		bus.Publish(c, bus.NewEvent(bus.Started, work.Repo, build, job))
	} else {
		bus.Publish(c, bus.NewEvent(bus.Finished, work.Repo, build, job))
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

	key := c.Param("id")
	logrus.Infof("Agent %s creating stream %s.", c.ClientIP(), key)

	wc, err := stream.Writer(c, key)
	if err != nil {
		c.String(500, "Failed to create stream writer. %s", err)
		return
	}

	defer func() {
		wc.Close()
		stream.Delete(c, key)
	}()

	io.Copy(wc, c.Request.Body)

	rc, err := stream.Reader(c, key)
	if err != nil {
		c.String(500, "Failed to create stream reader. %s", err)
		return
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer recover()
		store.WriteLog(c, &model.Job{ID: id}, rc)
		wg.Done()
	}()

	wc.Close()
	wg.Wait()
	c.String(200, "")

	logrus.Debugf("Agent %s wrote stream to database", c.ClientIP())
}

func Ping(c *gin.Context) {
	agent, err := store.GetAgentAddr(c, c.ClientIP())
	if err == nil {
		agent.Updated = time.Now().Unix()
		err = store.UpdateAgent(c, agent)
	} else {
		err = store.CreateAgent(c, &model.Agent{
			Address:  c.ClientIP(),
			Platform: "linux/amd64",
			Capacity: 2,
			Created:  time.Now().Unix(),
			Updated:  time.Now().Unix(),
		})
	}
	if err != nil {
		logrus.Errorf("Unable to register agent. %s", err.Error())
	}
	c.String(200, "PONG")
}

//
//
// Below are alternate implementations for the Queue that use websockets.
//
//

// PostLogs handles an http request from the agent to post build logs. These
// logs are posted at the end of the build process.
func PostLogs(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	job, err := store.GetJob(c, id)
	if err != nil {
		c.String(404, "Cannot upload logs. %s", err)
		return
	}
	if err := store.WriteLog(c, job, c.Request.Body); err != nil {
		c.String(500, "Cannot persist logs", err)
		return
	}
	c.String(200, "")
}

// WriteLogs handles an http request from the agent to stream build logs from
// the agent to the server to enable real time streamings to the client.
func WriteLogs(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(500, "Invalid input. %s", err)
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.String(500, "Cannot upgrade to websocket. %s", err)
		return
	}
	defer conn.Close()

	wc, err := stream.Writer(c, stream.ToKey(id))
	if err != nil {
		c.String(500, "Cannot create stream writer. %s", err)
		return
	}
	defer func() {
		wc.Close()
		stream.Delete(c, stream.ToKey(id))
	}()

	var msg []byte
	for {
		_, msg, err = conn.ReadMessage()
		if err != nil {
			break
		}
		wc.Write(msg)
		wc.Write(newline)
	}

	if err != nil && err != io.EOF {
		c.String(500, "Error reading logs. %s", err)
		return
	}
	//
	// rc, err := stream.Reader(c, stream.ToKey(id))
	// if err != nil {
	// 	c.String(500, "Failed to create stream reader. %s", err)
	// 	return
	// }
	//
	// wg := sync.WaitGroup{}
	// wg.Add(1)
	//
	// go func() {
	// 	defer recover()
	// 	store.WriteLog(c, &model.Job{ID: id}, rc)
	// 	wg.Done()
	// }()
	//
	// wc.Close()
	// wg.Wait()

}

// newline defines a newline constant to separate lines in the build output
var newline = []byte{'\n'}

// upgrader defines the default behavior for upgrading the websocket.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
