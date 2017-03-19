package compiler

import (
	"fmt"
	"path"

	"github.com/cncd/pipeline/pipeline/backend"
	"github.com/cncd/pipeline/pipeline/frontend/yaml"
)

func (c *Compiler) createProcess(name string, container *yaml.Container) *backend.Step {
	var (
		detached   bool
		workingdir string

		workspace  = fmt.Sprintf("%s_default:%s", c.prefix, c.base)
		privileged = container.Privileged
		entrypoint = container.Entrypoint
		command    = container.Command
		image      = expandImage(container.Image)
		// network    = container.Network
	)

	networks := []backend.Conn{
		backend.Conn{
			Name:    fmt.Sprintf("%s_default", c.prefix),
			Aliases: c.aliases,
		},
	}

	var volumes []string
	if !c.local {
		volumes = append(volumes, workspace)
	}
	volumes = append(volumes, c.volumes...)
	for _, volume := range container.Volumes.Volumes {
		volumes = append(volumes, volume.String())
	}
	// if network == "" {
	// 	network = fmt.Sprintf("%s_default", c.prefix)
	// 	for _, alias := range c.aliases {
	// 		// if alias != container.Name {
	// 		aliases = append(aliases, alias)
	// 		// }
	// 	}
	// } // host, bridge, none, container:<name>, overlay

	// append default environment variables
	environment := map[string]string{}
	for k, v := range container.Environment {
		environment[k] = v
	}
	for k, v := range c.env {
		switch v {
		case "", "0", "false":
			continue
		default:
			environment[k] = v
		}
	}

	environment["CI_WORKSPACE"] = path.Join(c.base, c.path)
	environment["DRONE_WORKSPACE"] = path.Join(c.base, c.path)

	if !isService(container) {
		workingdir = path.Join(c.base, c.path)
	}

	if isService(container) {
		detached = true
	}

	if isPlugin(container) {
		paramsToEnv(container.Vargs, environment)

		if imageMatches(container.Image, c.escalated) {
			privileged = true
			entrypoint = []string{}
			command = []string{}
		}
	}

	if isShell(container) {
		entrypoint = []string{"/bin/sh", "-c"}
		command = []string{"echo $CI_SCRIPT | base64 -d | /bin/sh -e"}
		environment["CI_SCRIPT"] = generateScriptPosix(container.Commands)
		environment["HOME"] = "/root"
		environment["SHELL"] = "/bin/sh"
	}

	return &backend.Step{
		Name:         name,
		Alias:        container.Name,
		Image:        image,
		Pull:         container.Pull,
		Detached:     detached,
		Privileged:   privileged,
		WorkingDir:   workingdir,
		Environment:  environment,
		Labels:       container.Labels,
		Entrypoint:   entrypoint,
		Command:      command,
		ExtraHosts:   container.ExtraHosts,
		Volumes:      volumes,
		Devices:      container.Devices,
		Networks:     networks,
		DNS:          container.DNS,
		DNSSearch:    container.DNSSearch,
		MemSwapLimit: int64(container.MemSwapLimit),
		MemLimit:     int64(container.MemLimit),
		ShmSize:      int64(container.ShmSize),
		CPUQuota:     int64(container.CPUQuota),
		CPUShares:    int64(container.CPUShares),
		CPUSet:       container.CPUSet,
		AuthConfig: backend.Auth{
			Username: container.AuthConfig.Username,
			Password: container.AuthConfig.Password,
			Email:    container.AuthConfig.Email,
		},
		OnSuccess: container.Constraints.Status.Match("success"),
		OnFailure: (len(container.Constraints.Status.Include)+
			len(container.Constraints.Status.Exclude) != 0) &&
			container.Constraints.Status.Match("failure"),
	}
}

func imageMatches(image string, to []string) bool {
	image = trimImage(image)
	for _, i := range to {
		if image == i {
			return true
		}
	}
	return false
}

func isPlugin(c *yaml.Container) bool {
	return len(c.Vargs) != 0
}

func isShell(c *yaml.Container) bool {
	return len(c.Commands) != 0
}

func isService(c *yaml.Container) bool {
	return c.Detached || (isPlugin(c) == false && isShell(c) == false)
}
