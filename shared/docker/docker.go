package docker

import (
	"errors"

	log "github.com/Sirupsen/logrus"
	"github.com/samalba/dockerclient"
)

var (
	LogOpts = &dockerclient.LogOptions{
		Stdout: true,
		Stderr: true,
	}

	LogOptsTail = &dockerclient.LogOptions{
		Follow: true,
		Stdout: true,
		Stderr: true,
	}
)

// Run creates the docker container, pulling images if necessary, starts
// the container and blocks until the container exits, returning the exit
// information.
func Run(client dockerclient.Client, conf *dockerclient.ContainerConfig, name string) (*dockerclient.ContainerInfo, error) {
	info, err := RunDaemon(client, conf, name)
	if err != nil {
		return nil, err
	}

	return Wait(client, info.Id)
}

// RunDaemon creates the docker container, pulling images if necessary, starts
// the container and returns the container information. It does not wait for
// the container to exit.
func RunDaemon(client dockerclient.Client, conf *dockerclient.ContainerConfig, name string) (*dockerclient.ContainerInfo, error) {

	// attempts to create the container
	id, err := client.CreateContainer(conf, name, nil)
	if err != nil {
		// and pull the image and re-create if that fails
		err = client.PullImage(conf.Image, nil)
		if err != nil {
			return nil, err
		}
		id, err = client.CreateContainer(conf, name, nil)
		if err != nil {
			client.RemoveContainer(id, true, true)
			return nil, err
		}
	}

	// fetches the container information
	info, err := client.InspectContainer(id)
	if err != nil {
		client.RemoveContainer(id, true, true)
		return nil, err
	}

	// starts the container
	err = client.StartContainer(id, &conf.HostConfig)
	if err != nil {
		client.RemoveContainer(id, true, true)
		return nil, err
	}

	return info, err
}

// Wait blocks until the named container exits, returning the exit information.
func Wait(client dockerclient.Client, name string) (*dockerclient.ContainerInfo, error) {

	defer func() {
		client.StopContainer(name, 5)
		client.KillContainer(name, "9")
	}()

	for attempts := 0; attempts < 5; attempts++ {
		done := client.Wait(name)
		<-done

		info, err := client.InspectContainer(name)
		if err != nil {
			return nil, err
		}

		if !info.State.Running {
			return info, nil
		}

		log.Debugf("attempting to resume waiting after %d attempts.\n", attempts)
	}

	return nil, errors.New("reached maximum wait attempts")
}
