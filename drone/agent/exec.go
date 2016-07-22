package agent

import (
	"bytes"
	"io/ioutil"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/drone/drone/agent"
	"github.com/drone/drone/build/docker"
	"github.com/drone/drone/client"

	"github.com/samalba/dockerclient"
)

type config struct {
	platform   string
	namespace  string
	privileged []string
	pull       bool
	logs       int64
	timeout    time.Duration
}

type pipeline struct {
	drone  client.Client
	docker dockerclient.Client
	config config
}

func (r *pipeline) run() error {
	w, err := r.drone.Pull("linux", "amd64")
	if err != nil {
		return err
	}
	running.Add(1)
	defer func() {
		running.Done()
	}()

	logrus.Infof("Starting build %s/%s#%d.%d",
		w.Repo.Owner, w.Repo.Name, w.Build.Number, w.Job.Number)

	cancel := make(chan bool, 1)
	engine := docker.NewClient(r.docker)

	// streaming the logs
	// rc, wc := io.Pipe()
	// defer func() {
	// 	wc.Close()
	// 	rc.Close()
	// }()

	var buf bytes.Buffer

	stream, err := r.drone.LogStream(w.Job.ID)
	if err != nil {
		return err
	}

	a := agent.Agent{
		Update: agent.NewClientUpdater(r.drone),
		// Logger:    agent.NewClientLogger(r.drone, w.Job.ID, rc, wc, r.config.logs),
		Logger:    agent.NewStreamLogger(stream, &buf, r.config.logs),
		Engine:    engine,
		Timeout:   r.config.timeout,
		Platform:  r.config.platform,
		Namespace: r.config.namespace,
		Escalate:  r.config.privileged,
		Pull:      r.config.pull,
	}

	// signal for canceling the build.
	wait := r.drone.Wait(w.Job.ID)
	defer wait.Cancel()
	go func() {
		if _, err := wait.Done(); err == nil {
			cancel <- true
			logrus.Infof("Cancel build %s/%s#%d.%d",
				w.Repo.Owner, w.Repo.Name, w.Build.Number, w.Job.Number)
		}
	}()

	a.Run(w, cancel)

	if err := r.drone.LogPost(w.Job.ID, ioutil.NopCloser(&buf)); err != nil {
		logrus.Errorf("Error sending logs for %s/%s#%d.%d",
			w.Repo.Owner, w.Repo.Name, w.Build.Number, w.Job.Number)
	}
	stream.Close()

	logrus.Infof("Finished build %s/%s#%d.%d",
		w.Repo.Owner, w.Repo.Name, w.Build.Number, w.Job.Number)

	return nil
}
