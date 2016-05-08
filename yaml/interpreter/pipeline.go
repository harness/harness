package interpreter

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/drone/drone/yaml"

	"github.com/samalba/dockerclient"
)

// element represents a link in the linked list.
type element struct {
	*yaml.Container
	next *element
}

// Pipeline represents a build pipeline.
type Pipeline struct {
	conf *yaml.Config
	head *element
	tail *element
	next chan (error)
	done chan (error)
	err  error

	containers []string
	volumes    []string
	networks   []string

	client dockerclient.Client
}

// Load loads the pipeline from the Yaml configuration file.
func Load(conf *yaml.Config) *Pipeline {
	pipeline := Pipeline{
		conf: conf,
		next: make(chan error),
		done: make(chan error),
	}

	var containers []*yaml.Container
	containers = append(containers, conf.Services...)
	containers = append(containers, conf.Pipeline...)

	for i, c := range containers {
		next := &element{Container: c}
		if i == 0 {
			pipeline.head = next
			pipeline.tail = next
		} else {
			pipeline.tail.next = next
			pipeline.tail = next
		}
	}

	go func() {
		pipeline.next <- nil
	}()

	return &pipeline
}

// Done returns when the process is done executing.
func (p *Pipeline) Done() <-chan error {
	return p.done
}

// Err returns the error for the current process.
func (p *Pipeline) Err() error {
	return p.err
}

// Next returns the next step in the process.
func (p *Pipeline) Next() <-chan error {
	return p.next
}

// Exec executes the current step.
func (p *Pipeline) Exec() {
	err := p.exec(p.head.Container)
	if err != nil {
		p.err = err
	}
	p.step()
}

// Skip skips the current step.
func (p *Pipeline) Skip() {
	p.step()
}

// Head returns the head item in the list.
func (p *Pipeline) Head() *yaml.Container {
	return p.head.Container
}

// Tail returns the tail item in the list.
func (p *Pipeline) Tail() *yaml.Container {
	return p.tail.Container
}

// Stop stops the pipeline.
func (p *Pipeline) Stop() {
	p.close(ErrTerm)
	return
}

// Setup prepares the build pipeline environment.
func (p *Pipeline) Setup() error {
	return nil
}

// Teardown removes the pipeline environment.
func (p *Pipeline) Teardown() {
	for _, id := range p.containers {
		p.client.StopContainer(id, 1)
		p.client.KillContainer(id, "9")
		p.client.RemoveContainer(id, true, true)
	}
	for _, id := range p.networks {
		p.client.RemoveNetwork(id)
	}
	for _, id := range p.volumes {
		p.client.RemoveVolume(id)
	}
}

// step steps through the pipeline to head.next
func (p *Pipeline) step() {
	if p.head == p.tail {
		p.close(nil)
		return
	}
	go func() {
		p.head = p.head.next
		p.next <- nil
	}()
}

// close closes open channels and signals the pipeline is done.
func (p *Pipeline) close(err error) {
	go func() {
		p.done <- nil
		close(p.next)
		close(p.done)
	}()
}

func (p *Pipeline) exec(c *yaml.Container) error {
	conf := toContainerConfig(c)
	auth := toAuthConfig(c)

	// check for the image and pull if not exists or if configured to always
	// pull the latest version.
	_, err := p.client.InspectImage(c.Image)
	if err == nil || c.Pull {
		err = p.client.PullImage(c.Image, auth)
		if err != nil {
			return err
		}
	}

	// creates and starts the container.
	id, err := p.client.CreateContainer(conf, c.ID, auth)
	if err != nil {
		return err
	}
	p.containers = append(p.containers, id)

	err = p.client.StartContainer(c.ID, &conf.HostConfig)
	if err != nil {
		return err
	}

	// stream the container logs
	go func() {
		rc, rerr := toLogs(p.client, c.ID)
		if rerr != nil {
			return
		}
		defer rc.Close()

		num := 0
		// now := time.Now().UTC()
		scanner := bufio.NewScanner(rc)
		for scanner.Scan() {
			// r.pipe.lines <- &Line{
			// 	Proc: c.Name,
			// 	Time: int64(time.Since(now).Seconds()),
			// 	Pos:  num,
			// 	Out:  scanner.Text(),
			// }
			num++
		}
	}()

	// if the container is run in detached mode we can exit without waiting
	// for execution to complete.
	if c.Detached {
		return nil
	}

	<-p.client.Wait(c.ID)

	res, err := p.client.InspectContainer(c.ID)
	if err != nil {
		return err
	}

	if res.State.OOMKilled {
		return &OomError{c.Name}
	} else if res.State.ExitCode != 0 {
		return &ExitError{c.Name, res.State.ExitCode}
	}
	return nil
}

func toLogs(client dockerclient.Client, id string) (io.ReadCloser, error) {
	opts := &dockerclient.LogOptions{
		Follow: true,
		Stdout: true,
		Stderr: true,
	}

	piper, pipew := io.Pipe()
	go func() {
		defer pipew.Close()

		// sometimes the docker logs fails due to parsing errors. this routine will
		// check for such a failure and attempt to resume if necessary.
		for i := 0; i < 5; i++ {
			if i > 0 {
				opts.Tail = 1
			}

			rc, err := client.ContainerLogs(id, opts)
			if err != nil {
				return
			}
			defer rc.Close()

			// use Docker StdCopy
			// internal.StdCopy(pipew, pipew, rc)

			// check to see if the container is still running. If not,  we can safely
			// exit and assume there are no more logs left to stream.
			v, err := client.InspectContainer(id)
			if err != nil || !v.State.Running {
				return
			}
		}
	}()
	return piper, nil
}

// helper function that converts the Continer data structure to the exepcted
// dockerclient.ContainerConfig.
func toContainerConfig(c *yaml.Container) *dockerclient.ContainerConfig {
	config := &dockerclient.ContainerConfig{
		Image:      c.Image,
		Env:        toEnvironmentSlice(c.Environment),
		Cmd:        c.Command,
		Entrypoint: c.Entrypoint,
		WorkingDir: c.WorkingDir,
		HostConfig: dockerclient.HostConfig{
			Privileged:       c.Privileged,
			NetworkMode:      c.Network,
			Memory:           c.MemLimit,
			CpuShares:        c.CPUShares,
			CpuQuota:         c.CPUQuota,
			CpusetCpus:       c.CPUSet,
			MemorySwappiness: -1,
			OomKillDisable:   c.OomKillDisable,
		},
	}

	if len(config.Entrypoint) == 0 {
		config.Entrypoint = nil
	}
	if len(config.Cmd) == 0 {
		config.Cmd = nil
	}
	if len(c.ExtraHosts) > 0 {
		config.HostConfig.ExtraHosts = c.ExtraHosts
	}
	if len(c.DNS) != 0 {
		config.HostConfig.Dns = c.DNS
	}
	if len(c.DNSSearch) != 0 {
		config.HostConfig.DnsSearch = c.DNSSearch
	}
	if len(c.VolumesFrom) != 0 {
		config.HostConfig.VolumesFrom = c.VolumesFrom
	}

	config.Volumes = map[string]struct{}{}
	for _, path := range c.Volumes {
		if strings.Index(path, ":") == -1 {
			config.Volumes[path] = struct{}{}
			continue
		}
		parts := strings.Split(path, ":")
		config.Volumes[parts[1]] = struct{}{}
		config.HostConfig.Binds = append(config.HostConfig.Binds, path)
	}

	for _, path := range c.Devices {
		if strings.Index(path, ":") == -1 {
			continue
		}
		parts := strings.Split(path, ":")
		device := dockerclient.DeviceMapping{
			PathOnHost:        parts[0],
			PathInContainer:   parts[1],
			CgroupPermissions: "rwm",
		}
		config.HostConfig.Devices = append(config.HostConfig.Devices, device)
	}

	return config
}

// helper function that converts the AuthConfig data structure to the exepcted
// dockerclient.AuthConfig.
func toAuthConfig(c *yaml.Container) *dockerclient.AuthConfig {
	if c.AuthConfig.Username == "" &&
		c.AuthConfig.Password == "" {
		return nil
	}
	return &dockerclient.AuthConfig{
		Email:    c.AuthConfig.Email,
		Username: c.AuthConfig.Username,
		Password: c.AuthConfig.Password,
	}
}

// helper function that converts a key value map of environment variables to a
// string slice in key=value format.
func toEnvironmentSlice(env map[string]string) []string {
	var envs []string
	for k, v := range env {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}
	return envs
}
