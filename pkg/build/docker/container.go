package docker

import (
	"fmt"
	"io"
)

type ContainerService struct {
	*Client
}

// List only running containers.
func (c *ContainerService) List() ([]*Containers, error) {
	containers := []*Containers{}
	err := c.do("GET", "/containers/json?all=0", nil, &containers)
	return containers, err
}

// List all containers
func (c *ContainerService) ListAll() ([]*Containers, error) {
	containers := []*Containers{}
	err := c.do("GET", "/containers/json?all=1", nil, &containers)
	return containers, err
}

// Create a Container
func (c *ContainerService) Create(conf *Config) (*Run, error) {
	run, err := c.create(conf)
	switch {
	// if no error, exit immediately
	case err == nil:
		return run, nil
	// if error we exit, unless it is
	// a NOT FOUND error, which means we just
	// need to download the Image from the center
	// image index
	case err != nil && err != ErrNotFound:
		return nil, err
	}

	// attempt to pull the image
	if err := c.Images.Pull(conf.Image); err != nil {
		return nil, err
	}

	// now that we have the image, re-try creation
	return c.create(conf)
}

func (c *ContainerService) create(conf *Config) (*Run, error) {
	run := Run{}
	err := c.do("POST", "/containers/create", conf, &run)
	return &run, err
}

// Start the container id
func (c *ContainerService) Start(id string, conf *HostConfig) error {
	return c.do("POST", fmt.Sprintf("/containers/%s/start", id), &conf, nil)
}

// Stop the container id
func (c *ContainerService) Stop(id string, timeout int) error {
	return c.do("POST", fmt.Sprintf("/containers/%s/stop?t=%v", id, timeout), nil, nil)
}

// Remove the container id from the filesystem.
func (c *ContainerService) Remove(id string) error {
	return c.do("DELETE", fmt.Sprintf("/containers/%s", id), nil, nil)
}

// Block until container id stops, then returns the exit code
func (c *ContainerService) Wait(id string) (*Wait, error) {
	wait := Wait{}
	err := c.do("POST", fmt.Sprintf("/containers/%s/wait", id), nil, &wait)
	return &wait, err
}

// Attach to the container to stream the stdout and stderr
func (c *ContainerService) Attach(id string, out io.Writer) error {
	path := fmt.Sprintf("/containers/%s/attach?&stream=1&stdout=1&stderr=1", id)
	return c.hijack("POST", path, false, out)
}

// Stop the container id
func (c *ContainerService) Inspect(id string) (*Container, error) {
	container := Container{}
	err := c.do("GET", fmt.Sprintf("/containers/%s/json", id), nil, &container)
	return &container, err
}

// Run the container
func (c *ContainerService) Run(conf *Config, host *HostConfig, out io.Writer) (*Wait, error) {
	// create the container from the image
	run, err := c.Create(conf)
	if err != nil {
		return nil, err
	}

	// attach to the container
	go func() {
		c.Attach(run.ID, out)
	}()

	// start the container
	if err := c.Start(run.ID, host); err != nil {
		return nil, err
	}

	// wait for the container to stop
	wait, err := c.Wait(run.ID)
	if err != nil {
		return nil, err
	}

	return wait, nil
}

// Run the container as a Daemon
func (c *ContainerService) RunDaemon(conf *Config, host *HostConfig) (*Run, error) {
	run, err := c.Create(conf)
	if err != nil {
		return nil, err
	}

	// start the container
	err = c.Start(run.ID, host)
	return run, err
}

func (c *ContainerService) RunDaemonPorts(image string, ports ...string) (*Run, error) {
	// setup configuration
	config := Config{Image: image}
	config.ExposedPorts = make(map[Port]struct{})

	// host configuration
	host := HostConfig{}
	host.PortBindings = make(map[Port][]PortBinding)

	// loop through and add ports
	for _, port := range ports {
		config.ExposedPorts[Port(port+"/tcp")] = struct{}{}
		host.PortBindings[Port(port+"/tcp")] = []PortBinding{{HostIp: "127.0.0.1", HostPort: ""}}
	}
	//127.0.0.1::%s
	//map[3306/tcp:{}] map[3306/tcp:[{127.0.0.1 }]]
	return c.RunDaemon(&config, &host)
}
