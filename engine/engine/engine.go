package engine

import (
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/drone/drone/bus"
	"github.com/drone/drone/engine/compiler"
	"github.com/drone/drone/engine/runner"
	"github.com/drone/drone/engine/runner/docker"
	"github.com/drone/drone/model"
	"github.com/drone/drone/queue"
	"github.com/drone/drone/store"
	"github.com/drone/drone/stream"
	"golang.org/x/net/context"
)

// Poll polls the build queue for build jobs.
func Poll(c context.Context) {
	for {
		pollRecover(c)
	}
}

func pollRecover(c context.Context) {
	defer recover()
	poll(c)
}

func poll(c context.Context) {
	w := queue.Pull(c)

	logrus.Infof("Starting build %s/%s#%d.%d",
		w.Repo.Owner, w.Repo.Name, w.Build.Number, w.Job.Number)

	rc, wc, err := stream.Create(c, stream.ToKey(w.Job.ID))
	if err != nil {
		logrus.Errorf("Error opening build stream %s/%s#%d.%d. %s",
			w.Repo.Owner, w.Repo.Name, w.Build.Number, w.Job.Number, err)
	}

	defer func() {
		wc.Close()
		rc.Close()
		stream.Remove(c, stream.ToKey(w.Job.ID))
	}()

	w.Job.Status = model.StatusRunning
	w.Job.Started = time.Now().Unix()

	quitc := make(chan bool, 1)
	eventc := make(chan *bus.Event, 1)
	bus.Subscribe(c, eventc)

	compile := compiler.New()
	compile.Transforms(nil)
	spec, err := compile.CompileString(w.Yaml)
	if err != nil {
		// TODO handle error
		logrus.Infof("Error compiling Yaml %s/%s#%d %s",
			w.Repo.Owner, w.Repo.Name, w.Build.Number, err.Error())
		return
	}

	defer func() {
		bus.Unsubscribe(c, eventc)
		quitc <- true
	}()

	ctx := context.TODO()
	ctx, cancel := context.WithCancel(ctx)

	// TODO store the started build in the database
	// TODO publish the started build
	store.UpdateJob(c, w.Job)
	//store.Write(c, w.Job, rc)
	bus.Publish(c, bus.NewEvent(bus.Started, w.Repo, w.Build, w.Job))

	conf := runner.Config{
		Engine: docker.FromContext(c),
	}

	run := conf.Runner(ctx, spec)
	run.Run()
	defer cancel()

	go func() {
		for {
			select {
			case event := <-eventc:
				if event.Type == bus.Cancelled && event.Job.ID == w.Job.ID {
					logrus.Infof("Cancel build %s/%s#%d.%d",
						w.Repo.Owner, w.Repo.Name, w.Build.Number, w.Job.Number)
					cancel()
				}
			case <-quitc:
				return
			}
		}
	}()

	pipe := run.Pipe()
	for {
		line := pipe.Next()
		if line == nil {
			break
		}
		fmt.Println(line)
	}

	err = run.Wait()

	// catch the build result
	if err != nil {
		w.Job.ExitCode = 255
	}
	if exitErr, ok := err.(*runner.ExitError); ok {
		w.Job.ExitCode = exitErr.Code
	}

	w.Job.Finished = time.Now().Unix()

	switch w.Job.ExitCode {
	case 128, 130:
		w.Job.Status = model.StatusKilled
	case 0:
		w.Job.Status = model.StatusSuccess
	default:
		w.Job.Status = model.StatusFailure
	}

	// store the finished build in the database
	logs, _, err := stream.Open(c, stream.ToKey(w.Job.ID))
	if err != nil {
		logrus.Errorf("Error reading build stream %s/%s#%d.%d",
			w.Repo.Owner, w.Repo.Name, w.Build.Number, w.Job.Number)
	}
	defer func() {
		if logs != nil {
			logs.Close()
		}
	}()
	if err := store.WriteLog(c, w.Job, logs); err != nil {
		logrus.Errorf("Error persisting build stream %s/%s#%d.%d",
			w.Repo.Owner, w.Repo.Name, w.Build.Number, w.Job.Number)
	}
	if logs != nil {
		logs.Close()
	}

	// TODO publish the finished build
	store.UpdateJob(c, w.Job)
	bus.Publish(c, bus.NewEvent(bus.Finished, w.Repo, w.Build, w.Job))

	logrus.Infof("Finished build %s/%s#%d.%d",
		w.Repo.Owner, w.Repo.Name, w.Build.Number, w.Job.Number)
}
