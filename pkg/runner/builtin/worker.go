package builtin

import (
	"errors"
	"io"
	"io/ioutil"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/samalba/dockerclient"
	"github.com/drone/drone/pkg/types"
)

var ErrLogging = errors.New("Logs not available")

var (
	// options to fetch the stdout and stderr logs
	logOpts = &dockerclient.LogOptions{
		Stdout: true,
		Stderr: true,
	}

	// options to fetch the stdout and stderr logs
	// by tailing the output.
	logOptsTail = &dockerclient.LogOptions{
		Follow: true,
		Stdout: true,
		Stderr: true,
	}
)

var (
	// name of the build agent container.
	DefaultAgent = "drone/drone-exec:latest"

	// default name of the build agent executable
	DefaultEntrypoint = []string{"/bin/drone-build"}

	// default argument to invoke build steps
	DefaultBuildArgs = []string{"--cache", "--clone", "--build", "--deploy"}

	// default argument to invoke build steps
	DefaultPullRequestArgs = []string{"--cache", "--clone", "--build"}

	// default arguments to invoke notify steps
	DefaultNotifyArgs = []string{"--notify"}
)

type work struct {
	Repo      *types.Repo      `json:"repo"`
	Build     *types.Build     `json:"build"`
	Job       *types.Job       `json:"job"`
	System    *types.System    `json:"system"`
	Workspace *types.Workspace `json:"workspace"`
	Yaml      []byte           `json:"yaml"`

	// Keys    *types.Keypair `json:"keys"`        // remove
	// Netrc   *types.Netrc   `json:"netrc"`       // remove
	// Env     []string       `json:"environment"` // remove
	// Plugins []string       `json:"plugins"`     // remove
}

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
			"/drone":               struct{}{},
			"/var/run/docker.sock": struct{}{},
		},
	}

	// TEMPORARY: always try to pull the new image for now
	// since we'll be frequently updating the build image
	// for the next few weeks
	w.client.PullImage(conf.Image, nil)

	w.build, err = run(w.client, conf, name)
	if err != nil {
		return 1, err
	}
	return w.build.State.ExitCode, err
}

// Notify executes the notification steps.
func (w *worker) Notify(stdin []byte) error {
	// use the affinity parameter in case we are
	// using Docker swarm as a backend.
	environment := []string{"affinity:container==" + w.build.Id}

	// the build container is acting as an ambassador container
	// with a shared filesystem .
	volume := []string{w.build.Id}

	// the command line arguments passed into the
	// build agent container.
	args := DefaultNotifyArgs
	args = append(args, "--")
	args = append(args, string(stdin))

	conf := &dockerclient.ContainerConfig{
		Image:      DefaultAgent,
		Entrypoint: DefaultEntrypoint,
		Cmd:        args,
		Env:        environment,
		HostConfig: dockerclient.HostConfig{
			VolumesFrom: volume,
		},
	}

	var err error
	w.notify, err = run(w.client, conf, "")
	return err
}

// Logs returns a multi-reader that fetches the logs
// from the build and deploy agents.
func (w *worker) Logs() (io.ReadCloser, error) {
	if w.build == nil {
		return nil, ErrLogging
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

// run is a helper function that creates and starts a container,
// blocking until either complete.
func run(client dockerclient.Client, conf *dockerclient.ContainerConfig, name string) (*dockerclient.ContainerInfo, error) {

	// attempts to create the contianer
	id, err := client.CreateContainer(conf, name)
	if err != nil {
		// and pull the image and re-create if that fails
		client.PullImage(conf.Image, nil)
		id, err = client.CreateContainer(conf, name)
		// make sure the container is removed in
		// the event of a creation error.
		if err != nil && len(id) != 0 {
			client.RemoveContainer(id, true, true)
		}
		if err != nil {
			return nil, err
		}
	}

	// ensures the container is always stopped
	// and ready to be removed.
	defer func() {
		client.StopContainer(id, 5)
		client.KillContainer(id, "9")
	}()

	// fetches the container information.
	info, err := client.InspectContainer(id)
	if err != nil {
		return nil, err
	}

	// channel listening for errors while the
	// container is running async.
	errc := make(chan error, 1)
	infoc := make(chan *dockerclient.ContainerInfo, 1)
	go func() {

		// starts the container
		err := client.StartContainer(id, &conf.HostConfig)
		if err != nil {
			errc <- err
			return
		}

		// blocks and waits for the container to finish
		// by streaming the logs (to /dev/null). Ideally
		// we could use the `wait` function instead
		rc, err := client.ContainerLogs(id, logOptsTail)
		if err != nil {
			errc <- err
			return
		}
		io.Copy(ioutil.Discard, rc)
		rc.Close()

		// fetches the container information
		info, err := client.InspectContainer(id)
		if err != nil {
			errc <- err
			return
		}
		infoc <- info
	}()

	select {
	case info := <-infoc:
		return info, nil
	case err := <-errc:
		return info, err
	}
}
