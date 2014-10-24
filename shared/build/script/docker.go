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
