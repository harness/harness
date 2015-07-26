package builtin

import (
	"errors"
	"io"
	"time"
	"io/ioutil"

	"github.com/drone/drone/pkg/types"
	cluster_manager "github.com/drone/drone/pkg/cluster/builtin"
	"github.com/drone/drone/Godeps/_workspace/src/github.com/citadel/citadel"
	log "github.com/drone/drone/Godeps/_workspace/src/github.com/Sirupsen/logrus"
)

var (
	ErrTimeout = errors.New("Timeout")
	ErrLogging = errors.New("Logs not available")
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

	w.build, err = run(w.manager, image, name, w.timeout)
	if err != nil {
		return 1, err
	}
	containerInfo, err := w.build.Engine.Info(w.build)
	return containerInfo.State.ExitCode, err
}

// Notify executes the notification steps.
func (w *worker) Notify(stdin []byte) error {
	// the build container is acting as an ambassador container
	// with a shared filesystem .
	volume := []string{w.build.ID}

	// the command line arguments passed into the
	// build agent container.
	args := DefaultNotifyArgs
	args = append(args, "--")
	args = append(args, string(stdin))

	image := &citadel.Image{
		Type: "drone_internal",
		Name: DefaultAgent,
		Cpus: 0.5,
		Memory: 64,
		Entrypoint: DefaultEntrypoint,
		Args: args,
		VolumesFrom: volume,
	}

	var err error
	w.notify, err = run(w.manager, image, "", DefaultNotifyTimeout)
	return err
}

// Logs returns a multi-reader that fetches the logs
// from the build and deploy agents.
func (w *worker) Logs() (io.ReadCloser, error) {
	if w.build == nil {
		return nil, ErrLogging
	}
	return w.manager.Logs(w.build, false)
}

// Remove stops and removes the build, deploy and
// notification agents created for the build task.
func (w *worker) Remove() {
	if w.notify != nil {
		w.manager.RemoveContainer(w.notify)
	}
	if w.build != nil {
		w.manager.RemoveContainer(w.build)
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
		log.Errorf("Start container")
		container, err := manager.Start(image, true)
		if err != nil {
			log.Errorf("Error starting container: %s", err)
			errc <- err
			return
		}

		// blocks and waits for the container to finish
		// by streaming the logs (to /dev/null). Ideally
		// we could use the `wait` function instead
		rc, err := manager.Logs(container, true)
		if err != nil {
			errc <- err
			return
		}
		io.Copy(ioutil.Discard, rc)
		rc.Close()

		containerc <- container
	}()

	select {
	case container := <- containerc:
		log.Errorf("Stop container")
		manager.StopAndKillContainer(container)
		return container, nil
	case err := <- errc:
		return nil, err
	case <-time.After(timeout):
		container := manager.FindContainerByName(name)
		if container != nil {
			log.Errorf("Stop container by timeout")
			manager.StopAndKillContainer(container)
		}
		return nil, ErrTimeout
	}
}
