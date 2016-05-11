package docker

import (
	"io"

	"github.com/drone/drone/build"
	"github.com/drone/drone/build/docker/internal"
	"github.com/drone/drone/yaml"

	"github.com/samalba/dockerclient"
)

type dockerEngine struct {
	client dockerclient.Client
}

func (e *dockerEngine) ContainerStart(container *yaml.Container) (string, error) {
	conf := toContainerConfig(container)
	auth := toAuthConfig(container)

	// pull the image if it does not exists or if the Container
	// is configured to always pull a new image.
	_, err := e.client.InspectImage(container.Image)
	if err != nil || container.Pull {
		e.client.PullImage(container.Image, auth)
	}

	// create and start the container and return the Container ID.
	id, err := e.client.CreateContainer(conf, container.ID, auth)
	if err != nil {
		return id, err
	}
	err = e.client.StartContainer(id, &conf.HostConfig)
	if err != nil {

		// remove the container if it cannot be started
		e.client.RemoveContainer(id, true, true)
		return id, err
	}
	return id, nil
}

func (e *dockerEngine) ContainerStop(id string) error {
	e.client.StopContainer(id, 1)
	e.client.KillContainer(id, "9")
	return nil
}

func (e *dockerEngine) ContainerRemove(id string) error {
	e.client.StopContainer(id, 1)
	e.client.KillContainer(id, "9")
	e.client.RemoveContainer(id, true, true)
	return nil
}

func (e *dockerEngine) ContainerWait(id string) (*build.State, error) {
	// wait for the container to exit
	//
	// TODO(bradrydzewski) we should have a for loop here
	// to re-connect and wait if this channel returns a
	// result even though the container is still running.
	//
	<-e.client.Wait(id)
	v, err := e.client.InspectContainer(id)
	if err != nil {
		return nil, err
	}
	return &build.State{
		ExitCode:  v.State.ExitCode,
		OOMKilled: v.State.OOMKilled,
	}, nil
}

func (e *dockerEngine) ContainerLogs(id string) (io.ReadCloser, error) {
	opts := &dockerclient.LogOptions{
		Follow: true,
		Stdout: true,
		Stderr: true,
	}

	piper, pipew := io.Pipe()
	go func() {
		defer pipew.Close()

		// sometimes the docker logs fails due to parsing errors. this
		// routine will check for such a failure and attempt to resume
		// if necessary.
		for i := 0; i < 5; i++ {
			if i > 0 {
				opts.Tail = 1
			}

			rc, err := e.client.ContainerLogs(id, opts)
			if err != nil {
				return
			}
			defer rc.Close()

			// use Docker StdCopy
			internal.StdCopy(pipew, pipew, rc)

			// check to see if the container is still running. If not,
			// we can safely exit and assume there are no more logs left
			// to stream.
			v, err := e.client.InspectContainer(id)
			if err != nil || !v.State.Running {
				return
			}
		}
	}()
	return piper, nil
}
