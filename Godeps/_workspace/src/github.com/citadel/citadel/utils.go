package citadel

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/samalba/dockerclient"
)

type (
	ImageInfo struct {
		Name string
		Tag  string
	}
)

func parsePortInformation(info *dockerclient.ContainerInfo, c *Container) error {
	for pp, b := range info.NetworkSettings.Ports {
		parts := strings.Split(pp, "/")
		rawPort, proto := parts[0], parts[1]

		for _, binding := range b {
			port, err := strconv.Atoi(binding.HostPort)
			if err != nil {
				return err
			}

			containerPort, err := strconv.Atoi(rawPort)
			if err != nil {
				return err
			}
			c.Ports = append(c.Ports, &Port{
				HostIp:        binding.HostIp,
				Proto:         proto,
				Port:          port,
				ContainerPort: containerPort,
			})
		}
	}

	// if we are running in host network mode look at the exposed ports on the image
	// to find out what ports are being exposed
	if info.HostConfig.NetworkMode == "host" {
		for k := range info.Config.ExposedPorts {
			var (
				rawPort string

				parts = strings.Split(k, "/")
				proto = "tcp"
			)

			switch len(parts) {
			case 2:
				rawPort, proto = parts[0], parts[1]
			default:
				rawPort = parts[0]
			}

			port, err := strconv.Atoi(rawPort)
			if err != nil {
				return err
			}

			c.Ports = append(c.Ports, &Port{
				Proto: proto,
				Port:  port,
			})
		}
	}

	return nil
}

func FromDockerContainer(id, image string, engine *Engine) (*Container, error) {
	info, err := engine.client.InspectContainer(id)
	if err != nil {
		return nil, err
	}

	var (
		cType       = ""
		state       = "stopped"
		networkMode = "bridge"
		labels      = []string{}
		env         = make(map[string]string)
	)

	for _, e := range info.Config.Env {
		vals := strings.SplitN(e, "=", 2)
		k, v := vals[0], vals[1]

		switch k {
		case "_citadel_type":
			cType = v
		case "_citadel_labels":
			labels = strings.Split(v, ",")
		case "HOME", "DEBIAN_FRONTEND", "PATH":
			continue
		default:
			env[k] = v
		}
	}

	if info.State.Running {
		state = "running"
	}

	if m := info.HostConfig.NetworkMode; m != "" {
		networkMode = m
	}
	volDefs := info.Config.Volumes
	vols := []string{}
	for k, _ := range volDefs {
		vols = append(vols, k)
	}

	container := &Container{
		ID:     id,
		Engine: engine,
		Name:   info.Name,
		State:  state,
		Image: &Image{
			Name:        image,
			Cpus:        float64(info.Config.CpuShares) / 100.0 * engine.Cpus,
			Cpuset:      info.Config.Cpuset,
			Memory:      float64(info.Config.Memory / 1024 / 1024),
			Volumes:     vols,
			Environment: env,
			Entrypoint:  info.Config.Entrypoint,
			Hostname:    info.Config.Hostname,
			Domainname:  info.Config.Domainname,
			Type:        cType,
			Labels:      labels,
			NetworkMode: networkMode,
			Publish:     info.HostConfig.PublishAllPorts,
			Privileged:  info.HostConfig.Privileged,
			RestartPolicy: RestartPolicy{
				Name:              info.HostConfig.RestartPolicy.Name,
				MaximumRetryCount: info.HostConfig.RestartPolicy.MaximumRetryCount,
			},
		},
	}

	if err := parsePortInformation(info, container); err != nil {
		return nil, err
	}
	container.Image.BindPorts = container.Ports

	return container, nil
}

func ParseImageName(name string) *ImageInfo {
	imageInfo := &ImageInfo{
		Name: name,
		Tag:  "latest",
	}

	// image type (top-level / user)
	switch {
	case strings.Index(name, "/") == -1:
		if strings.Index(name, ":") != -1 {
			parts := strings.Split(name, ":")
			imageInfo.Name = parts[0]
			imageInfo.Tag = parts[1]
		}
	case strings.Index(name, "/") != -1:
		parts := strings.Split(name, "/")
		n := strings.Join(parts[:len(parts)-1], "/")
		t := parts[len(parts)-1]
		if strings.Index(t, ":") != -1 {
			tparts := strings.Split(t, ":")
			n = fmt.Sprintf("%s/%s", n, tparts[0])
			t = tparts[1]
		} else {
			n = name
			t = "latest"
		}
		imageInfo.Name = n
		imageInfo.Tag = t
	}
	return imageInfo
}
