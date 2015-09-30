package docker

import (
	"io"
	"io/ioutil"

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

	// attempts to create the contianer
	id, err := client.CreateContainer(conf, name)
	if err != nil {
		// and pull the image and re-create if that fails
		err = client.PullImage(conf.Image, nil)
		if err != nil {
			return nil, err
		}
		id, err = client.CreateContainer(conf, name)
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

	errc := make(chan error, 1)
	infoc := make(chan *dockerclient.ContainerInfo, 1)
	go func() {

		// blocks and waits for the container to finish
		// by streaming the logs (to /dev/null). Ideally
		// we could use the `wait` function instead
		rc, err := client.ContainerLogs(name, LogOptsTail)
		if err != nil {
			errc <- err
			return
		}
		io.Copy(ioutil.Discard, rc)
		rc.Close()

		info, err := client.InspectContainer(name)
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
		return nil, err
	}
}
