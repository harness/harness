package agent

import (
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/drone/drone/agent"
	"github.com/drone/drone/build/docker"
	"github.com/drone/drone/model"
	"github.com/drone/mq/stomp"

	"github.com/samalba/dockerclient"
)

type config struct {
	platform   string
	namespace  string
	privileged []string
	pull       bool
	logs       int64
	timeout    time.Duration
	extension  []string
}

type pipeline struct {
	drone  *stomp.Client
	docker dockerclient.Client
	config config
}

func (r *pipeline) run(w *model.Work) {

	// defer func() {
	// 	// r.drone.Ack(id, opts)
	// }()

	logrus.Infof("Starting build %s/%s#%d.%d",
		w.Repo.Owner, w.Repo.Name, w.Build.Number, w.Job.Number)

	cancel := make(chan bool, 1)
	engine := docker.NewClient(r.docker)

	a := agent.Agent{
		Update:    agent.NewClientUpdater(r.drone),
		Logger:    agent.NewClientLogger(r.drone, w.Job.ID, r.config.logs),
		Engine:    engine,
		Timeout:   r.config.timeout,
		Platform:  r.config.platform,
		Namespace: r.config.namespace,
		Escalate:  r.config.privileged,
		Extension: r.config.extension,
		Pull:      r.config.pull,
	}

	cancelFunc := func(m *stomp.Message) {
		defer m.Release()

		id := m.Header.GetInt64("job-id")
		if id == w.Job.ID {
			cancel <- true
			logrus.Infof("Cancel build %s/%s#%d.%d",
				w.Repo.Owner, w.Repo.Name, w.Build.Number, w.Job.Number)
		}
	}

	// signal for canceling the build.
	sub, err := r.drone.Subscribe("/topic/cancel", stomp.HandlerFunc(cancelFunc))
	if err != nil {
		logrus.Errorf("Error subscribing to /topic/cancel. %s", err)
	}
	defer func() {
		r.drone.Unsubscribe(sub)
	}()

	a.Run(w, cancel)

	// if err := r.drone.LogPost(w.Job.ID, ioutil.NopCloser(&buf)); err != nil {
	// 	logrus.Errorf("Error sending logs for %s/%s#%d.%d",
	// 		w.Repo.Owner, w.Repo.Name, w.Build.Number, w.Job.Number)
	// }
	// stream.Close()

	logrus.Infof("Finished build %s/%s#%d.%d",
		w.Repo.Owner, w.Repo.Name, w.Build.Number, w.Job.Number)
}
