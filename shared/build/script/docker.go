package script

const (
    DefaultDockerNetworkMode = "bridge"
    DefaultDockerUser = "root"
    DefaultDockerHome = "/root"
)

// Docker stores the configuration details for
// configuring docker container.
type Docker struct {
    // NetworkMode (also known as `--net` option)
    // Could be set only if Docker is running in privileged mode
    NetworkMode *string `yaml:"net,omitempty"`

	User *string `yaml:"user,omitempty"`
	Home *string `yaml:"home,omitempty"`
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

// Returns DefaultDockerUser
// when Docker.User is empty.
// Returns Docker.User
// when it is not empty.
func (d *Docker) GetUser() string {
    if d.User == nil {
        return DefaultDockerUser
    }
    return *d.User
}

// Returns DefaultDockerHome
// when Docker.Home and Docker.User is empty.
// Returns Docker.Home
// when it is not empty.
// Returns "/home/" + Docker.User
// when Docker.Home is empty and Docker.User is not empty.
func (d *Docker) GetHome() string {
    if d.Home == nil {
        if d.User == nil {
            return DefaultDockerHome
        } else {
            return "/home/" + (*d.User)
        }
    }
    return *d.Home
}
