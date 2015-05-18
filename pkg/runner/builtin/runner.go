package builtin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/drone/drone/pkg/docker"
	"github.com/drone/drone/pkg/queue"
	common "github.com/drone/drone/pkg/types"
	"github.com/samalba/dockerclient"

	log "github.com/Sirupsen/logrus"
)

var (
	// Defult docker host address
	DefaultHost = "unix:///var/run/docker.sock"

	// Docker host address from environment variable
	DockerHost = os.Getenv("DOCKER_HOST")
)

func init() {
	// if the environment doesn't specify a DOCKER_HOST
	// we should use the default Docker socket.
	if len(DockerHost) == 0 {
		DockerHost = DefaultHost
	}
}

type Runner struct {
	Updater
}

func (r *Runner) Run(w *queue.Work) error {
	var workers []*worker
	var client dockerclient.Client

	defer func() {
		recover()

		// ensures that all containers have been removed
		// from the host machine.
		for _, worker := range workers {
			worker.Remove()
		}

		// if any part of the commit fails and leaves
		// behind orphan sub-builds we need to cleanup
		// after ourselves.
		if w.Commit.State == common.StateRunning {
			// if any tasks are running or pending
			// we should mark them as complete.
			for _, b := range w.Commit.Builds {
				if b.State == common.StateRunning {
					b.State = common.StateError
					b.Finished = time.Now().UTC().Unix()
					b.Duration = b.Finished - b.Started
					b.ExitCode = 255
				}
				if b.State == common.StatePending {
					b.State = common.StateError
					b.Started = time.Now().UTC().Unix()
					b.Finished = time.Now().UTC().Unix()
					b.Duration = 0
					b.ExitCode = 255
				}
				r.SetBuild(w.Repo, w.Commit, b)
			}
			// must populate build start
			if w.Commit.Started == 0 {
				w.Commit.Started = time.Now().UTC().Unix()
			}
			// mark the build as complete (with error)
			w.Commit.State = common.StateError
			w.Commit.Finished = time.Now().UTC().Unix()
			r.SetCommit(w.User, w.Repo, w.Commit)
		}
	}()

	// marks the build as running
	w.Commit.Started = time.Now().UTC().Unix()
	w.Commit.State = common.StateRunning
	err := r.SetCommit(w.User, w.Repo, w.Commit)
	if err != nil {
		return err
	}

	// create the Docker client. In this version of Drone (alpha)
	// we do not spread builds across clients, but this can and
	// (probably) will change in the future.
	client, err = dockerclient.NewDockerClient(DockerHost, nil)
	if err != nil {
		return err
	}

	// loop through and execute the build and
	// clone steps for each build task.
	for _, task := range w.Commit.Builds {

		// marks the task as running
		task.State = common.StateRunning
		task.Started = time.Now().UTC().Unix()
		err = r.SetBuild(w.Repo, w.Commit, task)
		if err != nil {
			return err
		}

		work := &work{
			Repo:   w.Repo,
			Commit: w.Commit,
			Keys:   w.Keys,
			Netrc:  w.Netrc,
			Yaml:   w.Yaml,
			Build:  task,
		}
		in, err := json.Marshal(work)
		if err != nil {
			return err
		}

		worker := newWorkerTimeout(client, w.Repo.Timeout)
		workers = append(workers, worker)
		cname := cname(task)
		pullrequest := (w.Commit.PullRequest != "")
		state, builderr := worker.Build(cname, in, pullrequest)

		switch {
		case builderr == ErrTimeout:
			task.State = common.StateKilled
		case builderr != nil:
			task.State = common.StateError
		case state != 0:
			task.ExitCode = state
			task.State = common.StateFailure
		default:
			task.State = common.StateSuccess
		}

		// send the logs to the datastore
		var buf bytes.Buffer
		rc, err := worker.Logs()
		if err != nil && builderr != nil {
			buf.WriteString("001 Error launching build")
			buf.WriteString(builderr.Error())
		} else if err != nil {
			buf.WriteString("002 Error launching build")
			buf.WriteString(err.Error())
			return err
		} else {
			defer rc.Close()
			docker.StdCopy(&buf, &buf, rc)
		}
		err = r.SetLogs(w.Repo, w.Commit, task, ioutil.NopCloser(&buf))
		if err != nil {
			return err
		}

		// update the task in the datastore
		task.Finished = time.Now().UTC().Unix()
		task.Duration = task.Finished - task.Started
		err = r.SetBuild(w.Repo, w.Commit, task)
		if err != nil {
			return err
		}
	}

	// update the build state if any of the sub-tasks
	// had a non-success status
	w.Commit.State = common.StateSuccess
	for _, build := range w.Commit.Builds {
		if build.State != common.StateSuccess {
			w.Commit.State = build.State
			break
		}
	}
	err = r.SetCommit(w.User, w.Repo, w.Commit)
	if err != nil {
		return err
	}

	// loop through and execute the notifications and
	// the destroy all containers afterward.
	for i, build := range w.Commit.Builds {
		work := &work{
			Repo:   w.Repo,
			Commit: w.Commit,
			Keys:   w.Keys,
			Netrc:  w.Netrc,
			Yaml:   w.Yaml,
			Build:  build,
		}
		in, err := json.Marshal(work)
		if err != nil {
			return err
		}
		workers[i].Notify(in)
		break
	}

	return nil
}

func (r *Runner) Cancel(build *common.Build) error {
	client, err := dockerclient.NewDockerClient(DockerHost, nil)
	if err != nil {
		return err
	}
	return client.StopContainer(cname(build), 30)
}

func (r *Runner) Logs(build *common.Build) (io.ReadCloser, error) {
	client, err := dockerclient.NewDockerClient(DockerHost, nil)
	if err != nil {
		return nil, err
	}
	// make sure this container actually exists
	info, err := client.InspectContainer(cname(build))
	if err != nil {
		return nil, err
	}

	// verify the container is running. if not we'll
	// do an exponential backoff and attempt to wait
	if !info.State.Running {
		for i := 0; ; i++ {
			time.Sleep(1 * time.Second)
			info, err = client.InspectContainer(info.Id)
			if err != nil {
				return nil, err
			}
			if info.State.Running {
				break
			}
			if i == 5 {
				return nil, dockerclient.ErrNotFound
			}
		}
	}

	return client.ContainerLogs(info.Id, logOptsTail)
	// rc, err := client.ContainerLogs(info.Id, logOptsTail)
	// if err != nil {
	// 	return nil, err
	// }
	// pr, pw := io.Pipe()
	// go func() {
	// 	defer rc.Close()
	// 	docker.StdCopy(pw, pw, rc)
	// }()
	// return pr, nil
}

func cname(build *common.Build) string {
	return fmt.Sprintf("drone-%d", build.ID)
}

func (r *Runner) Poll(q queue.Queue) {
	for {
		w := q.Pull()
		q.Ack(w)
		err := r.Run(w)
		if err != nil {
			log.Error(err)
		}
	}
}
