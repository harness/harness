package builtin

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/drone/drone/common"
	"github.com/drone/drone/queue"
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

		// if any part of the build fails and leaves
		// behind orphan sub-builds we need to cleanup
		// after ourselves.
		if w.Build.State == common.StateRunning {
			// if any tasks are running or pending
			// we should mark them as complete.
			for _, t := range w.Build.Tasks {
				if t.State == common.StateRunning {
					t.State = common.StateError
					t.Finished = time.Now().UTC().Unix()
					t.Duration = t.Finished - t.Started
				}
				if t.State == common.StatePending {
					t.State = common.StateError
					t.Started = time.Now().UTC().Unix()
					t.Finished = time.Now().UTC().Unix()
					t.Duration = 0
				}
				r.SetTask(w.Repo, w.Build, t)
			}
			// must populate build start
			if w.Build.Started == 0 {
				w.Build.Started = time.Now().UTC().Unix()
			}
			// mark the build as complete (with error)
			w.Build.State = common.StateError
			w.Build.Finished = time.Now().UTC().Unix()
			w.Build.Duration = w.Build.Finished - w.Build.Started
			r.SetBuild(w.Repo, w.Build)
		}
	}()

	// marks the build as running
	w.Build.Started = time.Now().UTC().Unix()
	w.Build.State = common.StateRunning
	err := r.SetBuild(w.Repo, w.Build)
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
	for _, task := range w.Build.Tasks {

		// marks the task as running
		task.State = common.StateRunning
		task.Started = time.Now().UTC().Unix()
		err = r.SetTask(w.Repo, w.Build, task)
		if err != nil {
			return err
		}

		work := &work{
			Repo:  w.Repo,
			Build: w.Build,
			Keys:  w.Keys,
			Yaml:  w.Yaml,
			Task:  task,
		}
		in, err := json.Marshal(work)
		if err != nil {
			return err
		}
		worker := newWorkerTimeout(client, w.Repo.Timeout+10) // 10 minute buffer
		workers = append(workers, worker)
		cname := cname(w.Repo.FullName, w.Build.Number, task.Number)
		state, builderr := worker.Build(cname, in)

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
			buf.WriteString(builderr.Error())
		} else if err != nil {
			buf.WriteString(err.Error())
			return err
		} else {
			defer rc.Close()
			StdCopy(&buf, &buf, rc)
		}
		err = r.SetLogs(w.Repo, w.Build, task, ioutil.NopCloser(&buf))
		if err != nil {
			return err
		}

		// update the task in the datastore
		task.Finished = time.Now().UTC().Unix()
		task.Duration = task.Finished - task.Started
		err = r.SetTask(w.Repo, w.Build, task)
		if err != nil {
			return err
		}
	}

	// update the build state if any of the sub-tasks
	// had a non-success status
	w.Build.State = common.StateSuccess
	for _, task := range w.Build.Tasks {
		if task.State != common.StateSuccess {
			w.Build.State = task.State
			break
		}
	}
	err = r.SetBuild(w.Repo, w.Build)
	if err != nil {
		return err
	}

	// loop through and execute the notifications and
	// the destroy all containers afterward.
	for i, task := range w.Build.Tasks {
		work := &work{
			Repo:  w.Repo,
			Build: w.Build,
			Keys:  w.Keys,
			Yaml:  w.Yaml,
			Task:  task,
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

func (r *Runner) Cancel(repo string, build, task int) error {
	client, err := dockerclient.NewDockerClient(DockerHost, nil)
	if err != nil {
		return err
	}
	return client.StopContainer(cname(repo, build, task), 30)
}

func (r *Runner) Logs(repo string, build, task int) (io.ReadCloser, error) {
	client, err := dockerclient.NewDockerClient(DockerHost, nil)
	if err != nil {
		return nil, err
	}
	// make sure this container actually exists
	info, err := client.InspectContainer(cname(repo, build, task))
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

	rc, err := client.ContainerLogs(info.Id, logOptsTail)
	if err != nil {
		return nil, err
	}
	pr, pw := io.Pipe()
	go func() {
		defer rc.Close()
		StdCopy(pw, pw, rc)
	}()
	return pr, nil
}

func cname(repo string, number, task int) string {
	s := fmt.Sprintf("%s/%d/%d", repo, number, task)
	h := sha1.New()
	h.Write([]byte(s))
	hash := hex.EncodeToString(h.Sum(nil))[:10]
	return fmt.Sprintf("drone-%s", hash)
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
