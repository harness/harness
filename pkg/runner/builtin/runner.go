package builtin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"github.com/drone/drone/pkg/docker"
	"github.com/drone/drone/pkg/queue"
	"github.com/drone/drone/pkg/types"

	log "github.com/drone/drone/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	cluster "github.com/drone/drone/pkg/cluster/builtin"
)

type Runner struct {
	Updater
	Manager *cluster.Manager
}

func (r *Runner) Run(w *queue.Work) error {

	// defer func() {
	// 	recover()

	// 	// if any part of the commit fails and leaves
	// 	// behind orphan sub-builds we need to cleanup
	// 	// after ourselves.
	// 	if w.Build.Status == types.StateRunning {
	// 		// if any tasks are running or pending
	// 		// we should mark them as complete.
	// 		for _, b := range w.Build.Jobs {
	// 			if b.Status == types.StateRunning {
	// 				b.Status = types.StateError
	// 				b.Finished = time.Now().UTC().Unix()
	// 				b.ExitCode = 255
	// 			}
	// 			if b.Status == types.StatePending {
	// 				b.Status = types.StateError
	// 				b.Started = time.Now().UTC().Unix()
	// 				b.Finished = time.Now().UTC().Unix()
	// 				b.ExitCode = 255
	// 			}
	// 			r.SetJob(w.Repo, w.Build, b)
	// 		}
	// 		// must populate build start
	// 		if w.Build.Started == 0 {
	// 			w.Build.Started = time.Now().UTC().Unix()
	// 		}
	// 		// mark the build as complete (with error)
	// 		w.Build.Status = types.StateError
	// 		w.Build.Finished = time.Now().UTC().Unix()
	// 		r.SetBuild(w.User, w.Repo, w.Build)
	// 	}
	// }()

	build := r.Updater.GetBuild(w.Job.BuildID)
	current_time := time.Now().UTC().Unix()
	// marks the build as running
	if build.Status == types.StatePending {
		build.Started = current_time
		build.Status = types.StateRunning
		err := r.SetBuild(w.User, w.Repo, build)
		if err != nil {
			log.Errorf("failure to set build. %s", err)
			return err
		}
	}
	// marks the job as running
	w.Job.Started = current_time
	w.Job.Status = types.StateRunning
	err := r.SetJob(w.Repo, build, w.Job)
	if err != nil {
		log.Errorf("failure to set job. %s", err)
		return err
	}

	pullrequest := (build.PullRequest != nil)

	work := &work{
		Repo:    w.Repo,
		Build:   build,
		Keys:    w.Keys,
		Netrc:   w.Netrc,
		Yaml:    w.Yaml,
		Job:     w.Job,
		Env:     w.Env,
		Plugins: w.Plugins,
	}
	in, err := json.Marshal(work)

	if err != nil {
		log.Errorf("failure to marshalise work. %s", err)
		return err
	}

	worker := newWorkerTimeout(r.Manager, w.Repo.Timeout)
	cname := cname(w)
	state, builderr := worker.Build(cname, in, pullrequest)

	switch {
	case builderr == ErrTimeout:
		w.Job.Status = types.StateKilled
	case builderr != nil:
		w.Job.Status = types.StateError
	case state != 0:
		w.Job.ExitCode = state
		w.Job.Status = types.StateFailure
	default:
		w.Job.Status = types.StateSuccess
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
	err = r.SetLogs(w.Repo, build, w.Job, ioutil.NopCloser(&buf))
	if err != nil {
		return err
	}

	// update the task in the datastore
	w.Job.Finished = time.Now().UTC().Unix()
	err = r.SetJob(w.Repo, build, w.Job)
	if err != nil {
		return err
	}

	// update the build state if any of the sub-tasks
	// had a non-success status
	// w.Build.Status = types.StateSuccess
	// for _, job := range w.Build.Jobs {
	// 	if job.Status != types.StateSuccess {
	// 		w.Build.Status = job.Status
	// 		break
	// 	}
	// }
	// err = r.SetBuild(w.User, w.Repo, w.Build)
	// if err != nil {
	// 	return err
	// }

	// // loop through and execute the notifications and
	// // the destroy all containers afterward.
	// for i, job := range w.Build.Jobs {
	// 	work := &work{
	// 		Repo:    w.Repo,
	// 		Build:   w.Build,
	// 		Keys:    w.Keys,
	// 		Netrc:   w.Netrc,
	// 		Yaml:    w.Yaml,
	// 		Job:     job,
	// 		Env:     w.Env,
	// 		Plugins: w.Plugins,
	// 	}
	// 	in, err := json.Marshal(work)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	workers[i].Notify(in)
	// 	break
	// }

	return nil
}

func (r *Runner) Cancel(job *types.Job) error {
	client, err := dockerclient.NewDockerClient(DockerHost, nil)
	if err != nil {
		return err
	}
	return client.StopContainer(cname(job), 30)
}

func (r *Runner) Logs(job *types.Job) (io.ReadCloser, error) {
	client, err := dockerclient.NewDockerClient(DockerHost, nil)
	if err != nil {
		return nil, err
	}
	// make sure this container actually exists
	info, err := client.InspectContainer(cname(job))
	if err != nil {
		return nil, err
	}

	// verify the container is running. if not we'll
	// do an exponential backoff and attempt to wait
	if !info.State.Running {
		for i := 0;; i++ {
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
}

func cname(w *queue.Work) string {
	return fmt.Sprintf("drone-%d-%d-%d", w.Repo.ID, w.Job.BuildID, w.Job.ID)
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
