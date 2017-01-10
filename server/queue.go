package server

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/net/context"

	"github.com/Sirupsen/logrus"
	"github.com/drone/drone/model"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/store"
	"github.com/drone/mq/stomp"
	"github.com/gorilla/websocket"
)

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

// HandleUpdate handles build updates from the agent and persists to the database.
func HandleUpdate(c context.Context, message *stomp.Message) {
	defer func() {
		message.Release()
		if r := recover(); r != nil {
			err := r.(error)
			logrus.Errorf("Panic recover: broker update handler: %s", err)
		}
	}()

	work := new(model.Work)
	if err := message.Unmarshal(work); err != nil {
		logrus.Errorf("Invalid input. %s", err)
		return
	}

	// TODO(bradrydzewski) it is really annoying that we have to do this lookup
	// and I'd prefer not to. The reason we do this is because the Build and Job
	// have fields that aren't serialized to json and would be reset to their
	// empty values if we just saved what was coming in the http.Request body.
	build, err := store.GetBuild(c, work.Build.ID)
	if err != nil {
		logrus.Errorf("Unable to find build. %s", err)
		return
	}
	job, err := store.GetJob(c, work.Job.ID)
	if err != nil {
		logrus.Errorf("Unable to find job. %s", err)
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
		logrus.Errorf("Unable to update job. %s", err)
		return
	}

	if ok {
		// get the user because we transfer the user form the server to agent
		// and back we lose the token which does not get serialized to json.
		user, uerr := store.GetUser(c, work.User.ID)
		if uerr != nil {
			logrus.Errorf("Unable to find user. %s", err)
			return
		}
		remote.Status(c, user, work.Repo, build,
			fmt.Sprintf("%s/%s/%d", work.System.Link, work.Repo.FullName, work.Build.Number))
	}

	client := stomp.MustFromContext(c)
	err = client.SendJSON("/topic/events", model.Event{
		Type: func() model.EventType {
			// HACK we don't even really care about the event type.
			// so we should just simplify how events are triggered.
			if job.Status == model.StatusRunning {
				return model.Started
			}
			return model.Finished
		}(),
		Repo:  *work.Repo,
		Build: *build,
		Job:   *job,
	},
		stomp.WithHeader("repo", work.Repo.FullName),
		stomp.WithHeader("private", strconv.FormatBool(work.Repo.IsPrivate)),
	)
	if err != nil {
		logrus.Errorf("Unable to publish to /topic/events. %s", err)
	}

	if job.Status == model.StatusRunning {
		return
	}

	var buf bytes.Buffer
	var sub []byte

	done := make(chan bool)
	dest := fmt.Sprintf("/topic/logs.%d", job.ID)
	sub, err = client.Subscribe(dest, stomp.HandlerFunc(func(m *stomp.Message) {
		defer m.Release()
		if m.Header.GetBool("eof") {
			done <- true
			return
		}
		buf.Write(m.Body)
		buf.WriteByte('\n')
	}))

	if err != nil {
		logrus.Errorf("Unable to read logs from broker. %s", err)
		return
	}

	defer func() {
		client.Send(dest, []byte{}, stomp.WithRetain("remove"))
		client.Unsubscribe(sub)
	}()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		logrus.Errorf("Unable to read logs from broker. Timeout. %s", err)
		return
	}

	if err := store.WriteLog(c, job, &buf); err != nil {
		logrus.Errorf("Unable to write logs to store. %s", err)
		return
	}
}
