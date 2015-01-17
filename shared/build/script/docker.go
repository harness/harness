package script

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
