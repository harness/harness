package docker

import (
	"fmt"
	"strings"

	"github.com/drone/drone/yaml"
	"github.com/samalba/dockerclient"
)

// helper function that converts the Continer data structure to the exepcted
// dockerclient.ContainerConfig.
func toContainerConfig(c *yaml.Container) *dockerclient.ContainerConfig {
	config := &dockerclient.ContainerConfig{
		Image:      c.Image,
		Env:        toEnvironmentSlice(c.Environment),
		Labels:     c.Labels,
		Cmd:        c.Command,
		Entrypoint: c.Entrypoint,
		WorkingDir: c.WorkingDir,
		HostConfig: dockerclient.HostConfig{
			Privileged:       c.Privileged,
			NetworkMode:      c.Network,
			Memory:           c.MemLimit,
			ShmSize:          c.ShmSize,
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
func toAuthConfig(container *yaml.Container) *dockerclient.AuthConfig {
	if container.AuthConfig.Username == "" &&
		container.AuthConfig.Password == "" {
		return nil
	}
	return &dockerclient.AuthConfig{
		Email:    container.AuthConfig.Email,
		Username: container.AuthConfig.Username,
		Password: container.AuthConfig.Password,
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
