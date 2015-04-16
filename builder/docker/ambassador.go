package docker

import (
	"errors"

	log "github.com/Sirupsen/logrus"
	"github.com/samalba/dockerclient"
)

var errNop = errors.New("Operation not supported")

// Ambassador is a wrapper around the Docker client that
// provides a shared volume and network for all containers.
type Ambassador struct {
	dockerclient.Client
	name string
}

// NewAmbassador creates an ambassador container and wraps the Docker
// client to inject the ambassador volume and network into containers.
func NewAmbassador(client dockerclient.Client) (_ *Ambassador, err error) {
	amb := &Ambassador{client, ""}

	conf := &dockerclient.ContainerConfig{}
	host := &dockerclient.HostConfig{}
	conf.Entrypoint = []string{"/bin/sleep"}
	conf.Cmd = []string{"86400"}
	conf.Image = "busybox"
	conf.Volumes = map[string]struct{}{}
	conf.Volumes["/drone"] = struct{}{}

	// creates the ambassador container
	amb.name, err = client.CreateContainer(conf, "")
	if err != nil {
		log.WithField("ambassador", conf.Image).Errorln(err)

		// on failure attempts to pull the image
		client.PullImage(conf.Image, nil)

		// then attempts to re-create the container
		amb.name, err = client.CreateContainer(conf, "")
		if err != nil {
			log.WithField("ambassador", conf.Image).Errorln(err)
			return nil, err
		}
	}
	err = client.StartContainer(amb.name, host)
	if err != nil {
		log.WithField("ambassador", conf.Image).Errorln(err)
	}
	return amb, err
}

// Destroy stops and deletes the ambassador container.
func (c *Ambassador) Destroy() error {
	c.Client.StopContainer(c.name, 5)
	c.Client.KillContainer(c.name, "9")
	return c.Client.RemoveContainer(c.name, true, true)
}

// CreateContainer creates a container.
func (c *Ambassador) CreateContainer(conf *dockerclient.ContainerConfig, name string) (string, error) {
	log.WithField("image", conf.Image).Infoln("create container")

	// add the affinity flag for swarm
	conf.Env = append(conf.Env, "affinity:container=="+c.name)

	id, err := c.Client.CreateContainer(conf, name)
	if err != nil {
		log.WithField("image", conf.Image).Errorln(err)
	}
	return id, err
}

// StartContainer starts a container. The ambassador volume
// is automatically linked. The ambassador network is linked
// iff a network mode is not already specified.
func (c *Ambassador) StartContainer(id string, conf *dockerclient.HostConfig) error {
	log.WithField("container", id).Debugln("start container")

	conf.VolumesFrom = append(conf.VolumesFrom, c.name)
	if len(conf.NetworkMode) == 0 {
		conf.NetworkMode = "container:" + c.name
	}
	err := c.Client.StartContainer(id, conf)
	if err != nil {
		log.WithField("container", id).Errorln(err)
	}
	return err
}

// StopContainer stops a container.
func (c *Ambassador) StopContainer(id string, timeout int) error {
	log.WithField("container", id).Debugln("stop container")
	err := c.Client.StopContainer(id, timeout)
	if err != nil {
		log.WithField("container", id).Errorln(err)
	}
	return err
}

// PullImage pulls an image.
func (c *Ambassador) PullImage(name string, auth *dockerclient.AuthConfig) error {
	log.WithField("image", name).Debugln("pull image")
	err := c.Client.PullImage(name, auth)
	if err != nil {
		log.WithField("image", name).Errorln(err)
	}
	return err
}
