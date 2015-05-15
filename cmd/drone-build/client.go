package main

import (
	"errors"
	"os"

	"github.com/samalba/dockerclient"
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

// client is a wrapper around the default Docker client
// that tracks all created containers ensures some default
// configurations are in place.
type client struct {
	dockerclient.Client
	info  *dockerclient.ContainerInfo
	names []string // names of created containers
}

func newClient(docker dockerclient.Client) (*client, error) {

	// creates an ambassador container
	conf := &dockerclient.ContainerConfig{}
	conf.HostConfig = dockerclient.HostConfig{}
	conf.Entrypoint = []string{"/bin/sleep"}
	conf.Cmd = []string{"86400"}
	conf.Image = "busybox"
	conf.Volumes = map[string]struct{}{}
	conf.Volumes["/drone"] = struct{}{}
	info, err := daemon(docker, conf, false)
	if err != nil {
		return nil, err
	}

	return &client{Client: docker, info: info}, nil
}

// CreateContainer creates a container and internally
// caches its container id.
func (c *client) CreateContainer(conf *dockerclient.ContainerConfig, name string) (string, error) {
	conf.Env = append(conf.Env, "affinity:container=="+c.info.Id)
	id, err := c.Client.CreateContainer(conf, name)
	if err == nil {
		c.names = append(c.names, id)
	}
	return id, err
}

// StartContainer starts a container and links to an
// ambassador container sharing the build machiens volume.
func (c *client) StartContainer(id string, conf *dockerclient.HostConfig) error {
	conf.VolumesFrom = append(conf.VolumesFrom, c.info.Id)
	if len(conf.NetworkMode) == 0 {
		conf.NetworkMode = "container:" + c.info.Id
	}
	return c.Client.StartContainer(id, conf)
}

// Destroy will terminate and destroy all containers that
// were created by this client.
func (c *client) Destroy() error {
	for _, id := range c.names {
		c.Client.KillContainer(id, "9")
		c.Client.RemoveContainer(id, true, true)
	}
	c.Client.KillContainer(c.info.Id, "9")
	return c.Client.RemoveContainer(c.info.Id, true, true)
}

func run(client dockerclient.Client, conf *dockerclient.ContainerConfig, pull bool) (*dockerclient.ContainerInfo, error) {
	// force-pull the image if specified.
	// TEMPORARY while we are in beta mode we should always re-pull drone plugins
	if pull { //|| strings.HasPrefix(conf.Image, "plugins/") {
		client.PullImage(conf.Image, nil)
	}

	// attempts to create the contianer
	id, err := client.CreateContainer(conf, "")
	if err != nil {
		// and pull the image and re-create if that fails
		client.PullImage(conf.Image, nil)
		id, err = client.CreateContainer(conf, "")
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
		client.RemoveContainer(id, true, true)
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
		defer rc.Close()
		StdCopy(os.Stdout, os.Stdout, rc)

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
		// TODO checkout net.Context and cancel
		// case <-time.After(timeout):
		// 	return info, ErrTimeout
	}
}

func daemon(client dockerclient.Client, conf *dockerclient.ContainerConfig, pull bool) (*dockerclient.ContainerInfo, error) {
	// force-pull the image
	if pull {
		client.PullImage(conf.Image, nil)
	}

	// attempts to create the contianer
	id, err := client.CreateContainer(conf, "")
	if err != nil {
		// and pull the image and re-create if that fails
		client.PullImage(conf.Image, nil)
		id, err = client.CreateContainer(conf, "")
		// make sure the container is removed in
		// the event of a creation error.
		if err != nil && len(id) != 0 {
			client.RemoveContainer(id, true, true)
		}
		if err != nil {
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
	return info, err
}
