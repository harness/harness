package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/cncd/logging"
	"github.com/cncd/pipeline/pipeline/rpc"
	"github.com/cncd/pubsub"
	"github.com/cncd/queue"
	"github.com/gin-gonic/gin"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/store"
)

// This file is a complete disaster because I'm trying to wedge in some
// experimental code. Please pardon our appearance during renovations.

var config = struct {
	pubsub pubsub.Publisher
	queue  queue.Queue
	logger logging.Log
	secret string
	host   string
}{
	pubsub.New(),
	queue.New(),
	logging.New(),
	os.Getenv("DRONE_SECRET"),
	os.Getenv("DRONE_HOST"),
}

func init() {
	config.pubsub.Create(context.Background(), "topic/events")
}

// func SetupRPC() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		c.Next()
// 	}
// }

func RPCHandler(c *gin.Context) {

	if secret := c.Request.Header.Get("Authorization"); secret != "Bearer "+config.secret {
		log.Printf("Unable to connect agent. Invalid authorization token %q does not match %q", secret, config.secret)
		c.String(401, "Unable to connect agent. Invalid authorization token")
		return
	}
	peer := RPC{
		remote: remote.FromContext(c),
		store:  store.FromContext(c),
		queue:  config.queue,
		pubsub: config.pubsub,
		logger: config.logger,
		host:   config.host,
	}
	rpc.NewServer(&peer).ServeHTTP(c.Writer, c.Request)
}

type RPC struct {
	remote remote.Remote
	queue  queue.Queue
	pubsub pubsub.Publisher
	logger logging.Log
	store  store.Store
	host   string
}

// Next implements the rpc.Next function
func (s *RPC) Next(c context.Context, filter rpc.Filter) (*rpc.Pipeline, error) {
	fn := func(task *queue.Task) bool {
		for k, v := range filter.Labels {
			if task.Labels[k] != v {
				return false
			}
		}
		return true
	}
	task, err := s.queue.Poll(c, fn)
	if err != nil {
		return nil, err
	} else if task == nil {
		return nil, nil
	}
	pipeline := new(rpc.Pipeline)
	err = json.Unmarshal(task.Data, pipeline)
	return pipeline, err
}

// Wait implements the rpc.Wait function
func (s *RPC) Wait(c context.Context, id string) error {
	return s.queue.Wait(c, id)
}

// Extend implements the rpc.Extend function
func (s *RPC) Extend(c context.Context, id string) error {
	return s.queue.Extend(c, id)
}

// Update implements the rpc.Update function
func (s *RPC) Update(c context.Context, id string, state rpc.State) error {
	jobID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return err
	}

	job, err := s.store.GetJob(jobID)
	if err != nil {
		log.Printf("error: cannot find job with id %d: %s", jobID, err)
		return err
	}

	build, err := s.store.GetBuild(job.BuildID)
	if err != nil {
		log.Printf("error: cannot find build with id %d: %s", job.BuildID, err)
		return err
	}

	repo, err := s.store.GetRepo(build.RepoID)
	if err != nil {
		log.Printf("error: cannot find repo with id %d: %s", build.RepoID, err)
		return err
	}

	if build.Status != model.StatusRunning {

	}

	job.Started = state.Started
	job.Finished = state.Finished
	job.ExitCode = state.ExitCode
	job.Status = model.StatusRunning
	job.Error = state.Error

	if build.Status == model.StatusPending {
		build.Started = job.Started
		build.Status = model.StatusRunning
		s.store.UpdateBuild(build)
	}

	log.Printf("pipeline: update %s: exited=%v, exit_code=%d", id, state.Exited, state.ExitCode)

	if state.Exited {

		job.Status = model.StatusSuccess
		if job.ExitCode != 0 || job.Error != "" {
			job.Status = model.StatusFailure
		}

		// save the logs
		var buf bytes.Buffer
		if serr := s.logger.Snapshot(context.Background(), id, &buf); serr != nil {
			log.Printf("error: snapshotting logs: %s", serr)
		}
		if werr := s.store.WriteLog(job, &buf); werr != nil {
			log.Printf("error: persisting logs: %s", werr)
		}

		// close the logger
		s.logger.Close(c, id)
		s.queue.Done(c, id)
	}

	// hackity hack
	cc := context.WithValue(c, "store", s.store)
	ok, uerr := store.UpdateBuildJob(cc, build, job)
	if uerr != nil {
		log.Printf("error: updating job: %s", uerr)
	}
	if ok {
		// get the user because we transfer the user form the server to agent
		// and back we lose the token which does not get serialized to json.
		user, uerr := s.store.GetUser(repo.UserID)
		if uerr != nil {
			logrus.Errorf("Unable to find user. %s", err)
		} else {
			s.remote.Status(user, repo, build,
				fmt.Sprintf("%s/%s/%d", s.host, repo.FullName, build.Number))
		}
	}

	message := pubsub.Message{}
	message.Data, _ = json.Marshal(model.Event{
		Type: func() model.EventType {
			// HACK we don't even really care about the event type.
			// so we should just simplify how events are triggered.
			// WTF was this being used for?????????????????????????
			if job.Status == model.StatusRunning {
				return model.Started
			}
			return model.Finished
		}(),
		Repo:  *repo,
		Build: *build,
		Job:   *job,
	})
	message.Labels = map[string]string{
		"repo":    repo.FullName,
		"private": strconv.FormatBool(repo.IsPrivate),
	}
	s.pubsub.Publish(c, "topic/events", message)
	log.Println("finish rpc.update")
	return nil
}

// Upload implements the rpc.Upload function
func (s *RPC) Upload(c context.Context, id, mime string, file io.Reader) error { return nil }

// Done implements the rpc.Done function
func (s *RPC) Done(c context.Context, id string) error { return nil }

// Log implements the rpc.Log function
func (s *RPC) Log(c context.Context, id string, line *rpc.Line) error {
	entry := new(logging.Entry)
	entry.Data, _ = json.Marshal(line)
	fmt.Println(string(entry.Data))
	s.logger.Write(c, id, entry)
	return nil
}
