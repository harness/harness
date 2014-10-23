package script

import "github.com/drone/drone/shared/build/docker"

const (
	DefaultDockerNetworkMode = "bridge"
)

type Image struct {
	Name     string
	Username string
	Password string
	Email    string
}

// Docker stores the configuration details for
// configuring docker container.
type Docker struct {
	// Net (also known as `--net` option)
	// Could be set only if Docker is running in privileged mode
	Net *string `yaml:"net,omitempty"`

	// Advanced image options
	Image Image
}

// Returns DefaultNetworkMode
// when Docker.NetworkMode is empty.
// Returns Docker.NetworkMode
// when it is not empty.
func (d *Docker) NetworkMode() string {
	if d.Net == nil {
		return DefaultDockerNetworkMode
	}
	return *d.Net
}

func (d *Docker) AuthConfig() docker.AuthConfiguration {
	return docker.AuthConfiguration{Username: d.Image.Username, Password: d.Image.Password, Email: d.Image.Email}
}
