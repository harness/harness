package builtin

import (
	"errors"
	"io"
	"io/ioutil"
	"time"

	"github.com/drone/drone/pkg/types"
	cluster_manager "github.com/drone/drone/pkg/cluster/builtin"
	"github.com/drone/drone/Godeps/_workspace/src/github.com/citadel/citadel"
)

var (
	ErrTimeout = errors.New("Timeout")
	ErrLogging = errors.New("Logs not available")
)

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
	DefaultAgent = "drone/drone-build:latest"

// default name of the build agent executable
	DefaultEntrypoint = []string{"/bin/drone-build"}

// default argument to invoke build steps
	DefaultBuildArgs = []string{"--build", "--clone", "--publish", "--deploy"}

// default argument to invoke build steps
	DefaultPullRequestArgs = []string{"--build", "--clone"}

// default arguments to invoke notify steps
	DefaultNotifyArgs = []string{"--notify"}

// default arguments to invoke notify steps
	DefaultNotifyTimeout = time.Minute * 5
)

type work struct {
	Repo    *types.Repo    `json:"repo"`
	Build   *types.Build   `json:"build"`
	Job     *types.Job     `json:"job"`
	Keys    *types.Keypair `json:"keys"`
	Netrc   *types.Netrc   `json:"netrc"`
	Yaml    []byte         `json:"yaml"`
	Env     []string       `json:"environment"`
	Plugins []string       `json:"plugins"`
}

type worker struct {
	timeout time.Duration
	manager *cluster_manager.Manager
	build   *citadel.Container
	notify  *citadel.Container
}

func newWorker(master *cluster_manager.Manager) *worker {
	return newWorkerTimeout(master, 60) // default 60 minute timeout
}

func newWorkerTimeout(manager *cluster_manager.Manager, timeout int64) *worker {
	return &worker{
		timeout: time.Duration(timeout) * time.Minute,
		manager:  manager,
	}
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
	image := &citadel.Image{
		Type: "drone_internal",
		ContainerName: name,
		Name: DefaultAgent,
		Cpus: 1.0,
		Entrypoint: DefaultEntrypoint,
		Args: args,
		Volumes: []string{
			"/drone",
			"/var/run/docker.sock:/var/run/docker.sock",
		},
	}

	container, err := run(w.manager, image, name, w.timeout)
	if err != nil {
		return container, err
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
	w.notify, err = run(w.client, conf, "", DefaultNotifyTimeout)
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
// blocking until either complete or the timeout is reached. If
// the timeout is reached an ErrTimeout is returned, else the
// container info is returned.
func run(manager *cluster_manager.Manager, image *citadel.Image, name string, timeout time.Duration) (*citadel.Container, error) {

	// channel listening for errors while the
	// container is running async.
	errc := make(chan error, 1)
	containerc := make(chan *citadel.Container, 1)
	go func() {
		// attempts to create the container
		container, err := manager.Start(image, true)
		containerc <- container
		errc <- err
	}()

	select {
	case container := <- containerc:
		err := <- errc
		manager.StopAndKillContainer(container)
		return container, err
	case <-time.After(timeout):
		return nil, ErrTimeout
	}
}
