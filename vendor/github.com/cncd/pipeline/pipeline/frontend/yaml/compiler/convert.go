package compiler

import (
	"fmt"
	"path"
	"strings"

	"github.com/cncd/pipeline/pipeline/backend"
	"github.com/cncd/pipeline/pipeline/frontend/yaml"
)

func (c *Compiler) createProcess(name string, container *yaml.Container) *backend.Step {
	var (
		detached   bool
		workingdir string

		workspace    = fmt.Sprintf("%s_default:%s", c.prefix, c.base)
		privileged   = container.Privileged
		entrypoint   = container.Entrypoint
		command      = container.Command
		image        = expandImage(container.Image)
		network_mode = container.NetworkMode
		// network    = container.Network
	)

	networks := []backend.Conn{
		backend.Conn{
			Name:    fmt.Sprintf("%s_default", c.prefix),
			Aliases: []string{container.Name},
		},
	}
	for _, network := range c.networks {
		networks = append(networks, backend.Conn{
			Name: network,
		})
	}

	var volumes []string
	if !c.local {
		volumes = append(volumes, workspace)
	}
	volumes = append(volumes, c.volumes...)
	for _, volume := range container.Volumes.Volumes {
		volumes = append(volumes, volume.String())
	}

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
	// TODO: This is here for backward compatibility and will eventually be removed.
	environment["DRONE_WORKSPACE"] = path.Join(c.base, c.path)

	if !isService(container) {
		workingdir = path.Join(c.base, c.path)
	}

	if isService(container) {
		detached = true
	}

	if isPlugin(container) {
		paramsToEnv(container.Vargs, environment)

		if matchImage(container.Image, c.escalated...) {
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

	authConfig := backend.Auth{
		Username: container.AuthConfig.Username,
		Password: container.AuthConfig.Password,
		Email:    container.AuthConfig.Email,
	}
	for _, registry := range c.registries {
		if matchHostname(image, registry.Hostname) {
			authConfig.Username = registry.Username
			authConfig.Password = registry.Password
			authConfig.Email = registry.Email
			break
		}
	}

	for _, requested := range container.Secrets.Secrets {
		secret, ok := c.secrets[strings.ToLower(requested.Source)]
		if ok && (len(secret.Match) == 0 || matchImage(image, secret.Match...)) {
			environment[strings.ToUpper(requested.Target)] = secret.Value
		}
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
		AuthConfig:   authConfig,
		OnSuccess:    container.Constraints.Status.Match("success"),
		OnFailure: (len(container.Constraints.Status.Include)+
			len(container.Constraints.Status.Exclude) != 0) &&
			container.Constraints.Status.Match("failure"),
		NetworkMode: network_mode,
	}
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
