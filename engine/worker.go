package engine

import (
	"fmt"
	"io"

	"github.com/drone/drone/shared/docker"
	"github.com/samalba/dockerclient"
)

var (
	// name of the build agent container.
	DefaultAgent = "drone/drone-exec:latest"

	// default name of the build agent executable
	DefaultEntrypoint = []string{"/bin/drone-exec"}

	// default argument to invoke build steps
	DefaultBuildArgs = []string{"--pull", "--cache", "--clone", "--build", "--deploy"}

	// default argument to invoke build steps
	DefaultPullRequestArgs = []string{"--pull", "--cache", "--clone", "--build"}

	// default arguments to invoke notify steps
	DefaultNotifyArgs = []string{"--pull", "--notify"}
)

type worker struct {
	client dockerclient.Client
	build  *dockerclient.ContainerInfo
	notify *dockerclient.ContainerInfo
}

func newWorker(client dockerclient.Client) *worker {
	return &worker{client: client}
}

// Build executes the clone, build and deploy steps.
func (w *worker) Build(name string, stdin []byte, pr bool) (_ int, err error) {
	// the command line arguments passed into the
	// build agent container.
	args := DefaultBuildArgs
	if pr {
		args = DefaultPullRequestArgs
	}
	args = append(args, "--")
	args = append(args, string(stdin))

	conf := &dockerclient.ContainerConfig{
		Image:      DefaultAgent,
		Entrypoint: DefaultEntrypoint,
		Cmd:        args,
		HostConfig: dockerclient.HostConfig{
			Binds: []string{"/var/run/docker.sock:/var/run/docker.sock"},
		},
		Volumes: map[string]struct{}{
			"/var/run/docker.sock": {},
		},
	}

	// TEMPORARY: always try to pull the new image for now
	// since we'll be frequently updating the build image
	// for the next few weeks
	w.client.PullImage(conf.Image, nil)

	w.build, err = docker.Run(w.client, conf, name)
	if err != nil {
		return 1, err
	}
	if w.build.State.OOMKilled {
		return 1, fmt.Errorf("OOMKill received")
	}
	return w.build.State.ExitCode, err
}

// Notify executes the notification steps.
func (w *worker) Notify(stdin []byte) error {

	args := DefaultNotifyArgs
	args = append(args, "--")
	args = append(args, string(stdin))

	conf := &dockerclient.ContainerConfig{
		Image:      DefaultAgent,
		Entrypoint: DefaultEntrypoint,
		Cmd:        args,
		HostConfig: dockerclient.HostConfig{},
	}

	var err error
	w.notify, err = docker.Run(w.client, conf, "")
	return err
}

// Logs returns a multi-reader that fetches the logs
// from the build and deploy agents.
func (w *worker) Logs() (io.ReadCloser, error) {
	if w.build == nil {
		return nil, errLogging
	}
	return w.client.ContainerLogs(w.build.Id, logOpts)
}

// Remove stops and removes the build, deploy and
// notification agents created for the build task.
func (w *worker) Remove() {
	if w.notify != nil {
		w.client.KillContainer(w.notify.Id, "9")
		w.client.RemoveContainer(w.notify.Id, true, true)
	}
	if w.build != nil {
		w.client.KillContainer(w.build.Id, "9")
		w.client.RemoveContainer(w.build.Id, true, true)
	}
}
