package script

import "strings"

const (
	DefaultDockerNetworkMode = "bridge"
)

// Docker stores the configuration details for
// configuring docker container.
type Docker struct {
	// NetworkMode (also known as `--net` option)
	// Could be set only if Docker is running in privileged mode
	NetworkMode *string `yaml:"net,omitempty"`

	// Hostname (also known as `--hostname` option)
	// Could be set only if Docker is running in privileged mode
	Hostname *string `yaml:"hostname,omitempty"`

	// Volumes lists a set of directorys that should
	// be a bound volume from the container host
	// to the container
	Volumes []string `yaml:"volumes"`
}

// DockerNetworkMode returns DefaultNetworkMode
// when Docker.NetworkMode is empty.
// DockerNetworkMode returns Docker.NetworkMode
// when it is not empty.
func DockerNetworkMode(d *Docker) string {
	if d == nil || d.NetworkMode == nil {
		return DefaultDockerNetworkMode
	}
	return *d.NetworkMode
}

// DockerNetworkMode returns empty string
// when Docker.NetworkMode is empty.
// DockerNetworkMode returns Docker.NetworkMode
// when it is not empty.
func DockerHostname(d *Docker) string {
	if d == nil || d.Hostname == nil {
		return ""
	}
	return *d.Hostname
}

// DockerVolumes returns empty []string
// when Docker.Volumes is empty.
func DockerVolumes(d *Docker) []string {
	if d == nil || d.Volumes == nil || len(d.Volumes) == 0 {
		return []string{}
	}
	return d.Volumes
}

// IsVolumeValid - given a volume string,
// we are going to extract the container mount and the host mount
// from the volume string, and return.  Will error if split fails
func IsVolumeValid(v string) bool {
	// split the volumes on ":"
	paths := strings.Split(v, ":")
	// something is wrong if there aren't 2 items in the slice
	if len(paths) != 2 {
		return false
	}
	// if the host or container path doesn't start with "/" not valid
	for _, path := range paths {
		if !strings.HasPrefix(path, "/") {
			return false
		}
	}
	return true
}
