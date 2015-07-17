package builtin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"
	"crypto/tls"
	"crypto/x509"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/samalba/dockerclient"
	"github.com/drone/drone/pkg/docker"
	"github.com/drone/drone/pkg/queue"
	"github.com/drone/drone/pkg/types"

	log "github.com/drone/drone/Godeps/_workspace/src/github.com/Sirupsen/logrus"
)

var (
	// Default docker host address
	DefaultHost = "unix:///var/run/docker.sock"

	// Docker host address from environment variable
	DockerHost = os.Getenv("DOCKER_HOST")

	// Docker TLS variables
	DockerHostCa = os.Getenv("DOCKER_CA")
	DockerHostKey = os.Getenv("DOCKER_KEY")
	DockerHostCert = os.Getenv("DOCKER_CERT")
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
	var tlc *tls.Config

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
		if w.Build.Status == types.StateRunning {
			// if any tasks are running or pending
			// we should mark them as complete.
			for _, b := range w.Build.Jobs {
				if b.Status == types.StateRunning {
					b.Status = types.StateError
					b.Finished = time.Now().UTC().Unix()
					b.ExitCode = 255
				}
				if b.Status == types.StatePending {
					b.Status = types.StateError
					b.Started = time.Now().UTC().Unix()
					b.Finished = time.Now().UTC().Unix()
					b.ExitCode = 255
				}
				r.SetJob(w.Repo, w.Build, b)
			}
			// must populate build start
			if w.Build.Started == 0 {
				w.Build.Started = time.Now().UTC().Unix()
			}
			// mark the build as complete (with error)
			w.Build.Status = types.StateError
			w.Build.Finished = time.Now().UTC().Unix()
			r.SetBuild(w.User, w.Repo, w.Build)
		}
	}()

	// marks the build as running
	w.Build.Started = time.Now().UTC().Unix()
	w.Build.Status = types.StateRunning
	err := r.SetBuild(w.User, w.Repo, w.Build)
	if err != nil {
		log.Errorf("failure to set build. %s", err)
		return err
	}

	// create the Docket client TLS config
	if len(DockerHostCert) > 0 && len(DockerHostKey) > 0 && len(DockerHostCa) > 0 {
		cert, err := tls.LoadX509KeyPair(DockerHostCert, DockerHostKey)
		if err != nil {
			log.Errorf("failure to load SSL cert and key. %s", err)
		}
		caCert, err := ioutil.ReadFile(DockerHostCa)
		if err != nil {
			log.Errorf("failure to load SSL CA cert. %s", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlc = &tls.Config{
			Certificates:       []tls.Certificate{cert},
			RootCAs:            caCertPool,
		}
	}

	// create the Docker client. In this version of Drone (alpha)
	// we do not spread builds across clients, but this can and
	// (probably) will change in the future.
	client, err = dockerclient.NewDockerClient(DockerHost, tlc)
	if err != nil {
		log.Errorf("failure to connect to docker. %s", err)
		return err
	}

	// loop through and execute the build and
	// clone steps for each build job.
	for _, job := range w.Build.Jobs {

		// marks the task as running
		job.Status = types.StateRunning
		job.Started = time.Now().UTC().Unix()
		err = r.SetJob(w.Repo, w.Build, job)
		if err != nil {
			log.Errorf("failure to set job. %s", err)
			return err
		}

		work := &work{
			Repo:    w.Repo,
			Build:   w.Build,
			Keys:    w.Keys,
			Netrc:   w.Netrc,
			Yaml:    w.Yaml,
			Job:     job,
			Env:     w.Env,
			Plugins: w.Plugins,
		}
		in, err := json.Marshal(work)
		if err != nil {
			log.Errorf("failure to marshalise work. %s", err)
			return err
		}

		worker := newWorkerTimeout(client, w.Repo.Timeout)
		workers = append(workers, worker)
		cname := cname(job)
		pullrequest := (w.Build.PullRequest != nil)
		state, builderr := worker.Build(cname, in, pullrequest)

		switch {
		case builderr == ErrTimeout:
			job.Status = types.StateKilled
		case builderr != nil:
			job.Status = types.StateError
		case state != 0:
			job.ExitCode = state
			job.Status = types.StateFailure
		default:
			job.Status = types.StateSuccess
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
		err = r.SetLogs(w.Repo, w.Build, job, ioutil.NopCloser(&buf))
		if err != nil {
			return err
		}

		// update the task in the datastore
		job.Finished = time.Now().UTC().Unix()
		err = r.SetJob(w.Repo, w.Build, job)
		if err != nil {
			return err
		}
	}

	// update the build state if any of the sub-tasks
	// had a non-success status
	w.Build.Status = types.StateSuccess
	for _, job := range w.Build.Jobs {
		if job.Status != types.StateSuccess {
			w.Build.Status = job.Status
			break
		}
	}
	err = r.SetBuild(w.User, w.Repo, w.Build)
	if err != nil {
		return err
	}

	// loop through and execute the notifications and
	// the destroy all containers afterward.
	for i, job := range w.Build.Jobs {
		work := &work{
			Repo:    w.Repo,
			Build:   w.Build,
			Keys:    w.Keys,
			Netrc:   w.Netrc,
			Yaml:    w.Yaml,
			Job:     job,
			Env:     w.Env,
			Plugins: w.Plugins,
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
}

func cname(job *types.Job) string {
	return fmt.Sprintf("drone-%d", job.ID)
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
