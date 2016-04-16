package runner

import "fmt"

// Container defines the container configuration.
type Container struct {
	Name           string            `json:"name"`
	Alias          string            `json:"alias"`
	Image          string            `json:"image"`
	Pull           bool              `json:"pull,omitempty"`
	AuthConfig     Auth              `json:"auth_config,omitempty"`
	Privileged     bool              `json:"privileged,omitempty"`
	WorkingDir     string            `json:"working_dir,omitempty"`
	Environment    map[string]string `json:"environment,omitempty"`
	Entrypoint     []string          `json:"entrypoint,omitempty"`
	Command        []string          `json:"command,omitempty"`
	ExtraHosts     []string          `json:"extra_hosts,omitempty"`
	Volumes        []string          `json:"volumes,omitempty"`
	VolumesFrom    []string          `json:"volumes_from,omitempty"`
	Devices        []string          `json:"devices,omitempty"`
	Network        string            `json:"network_mode,omitempty"`
	DNS            []string          `json:"dns,omitempty"`
	DNSSearch      []string          `json:"dns_search,omitempty"`
	MemSwapLimit   int64             `json:"memswap_limit,omitempty"`
	MemLimit       int64             `json:"mem_limit,omitempty"`
	CPUQuota       int64             `json:"cpu_quota,omitempty"`
	CPUShares      int64             `json:"cpu_shares,omitempty"`
	CPUSet         string            `json:"cpuset,omitempty"`
	OomKillDisable bool              `json:"oom_kill_disable,omitempty"`
}

// Validate validates the container configuration details and returns an error
// if the validation fails.
func (c *Container) Validate() error {
	switch {

	case c.Name == "":
		return fmt.Errorf("Missing container name")
	case c.Image == "":
		return fmt.Errorf("Missing container image")
	default:
		return nil
	}

}

// Auth provides authentication parameters to authenticate to a remote
// container registry for image download.
type Auth struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Email    string `json:"email,omitempty"`
	Token    string `json:"registry_token,omitempty"`
}

// Volume defines a container volume.
type Volume struct {
	Name       string            `json:"name,omitempty"`
	Alias      string            `json:"alias,omitempty"`
	Driver     string            `json:"driver,omitempty"`
	DriverOpts map[string]string `json:"driver_opts,omitempty"`
	External   bool              `json:"external,omitempty"`
}

// Network defines a container network.
type Network struct {
	Name       string            `json:"name,omitempty"`
	Alias      string            `json:"alias,omitempty"`
	Driver     string            `json:"driver,omitempty"`
	DriverOpts map[string]string `json:"driver_opts,omitempty"`
	External   bool              `json:"external,omitempty"`
}
