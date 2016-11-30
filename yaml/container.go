package yaml

import (
	"fmt"

	"github.com/drone/drone/yaml/types"
	"gopkg.in/yaml.v2"
)

// Auth defines Docker authentication credentials.
type Auth struct {
	Username string
	Password string
	Email    string
}

// Container defines a Docker container.
type Container struct {
	ID             string
	Name           string
	Image          string
	Build          string
	Pull           bool
	AuthConfig     Auth
	Detached       bool
	Disabled       bool
	Privileged     bool
	WorkingDir     string
	Environment    map[string]string
	Labels         map[string]string
	Entrypoint     []string
	Command        []string
	Commands       []string
	ExtraHosts     []string
	Volumes        []string
	VolumesFrom    []string
	Devices        []string
	Network        string
	DNS            []string
	DNSSearch      []string
	MemSwapLimit   int64
	MemLimit       int64
	ShmSize        int64
	CPUQuota       int64
	CPUShares      int64
	CPUSet         string
	OomKillDisable bool
	Constraints    Constraints

	Vargs map[string]interface{}
}

// container is an intermediate type used for decoding a container in a format
// compatible with docker-compose.yml.

// this file has a bunch of custom types that are annoying to work with, which
// is why this is used for intermediate purposes and converted to something
// easier to work with.
type container struct {
	Name           string              `yaml:"name"`
	Image          string              `yaml:"image"`
	Build          string              `yaml:"build"`
	Pull           bool                `yaml:"pull"`
	Detached       bool                `yaml:"detach"`
	Privileged     bool                `yaml:"privileged"`
	Environment    types.MapEqualSlice `yaml:"environment"`
	Labels         types.MapEqualSlice `yaml:"labels"`
	Entrypoint     types.StringOrSlice `yaml:"entrypoint"`
	Command        types.StringOrSlice `yaml:"command"`
	Commands       types.StringOrSlice `yaml:"commands"`
	ExtraHosts     types.StringOrSlice `yaml:"extra_hosts"`
	Volumes        types.StringOrSlice `yaml:"volumes"`
	VolumesFrom    types.StringOrSlice `yaml:"volumes_from"`
	Devices        types.StringOrSlice `yaml:"devices"`
	Network        string              `yaml:"network_mode"`
	DNS            types.StringOrSlice `yaml:"dns"`
	DNSSearch      types.StringOrSlice `yaml:"dns_search"`
	MemSwapLimit   int64               `yaml:"memswap_limit"`
	MemLimit       int64               `yaml:"mem_limit"`
	ShmSize        int64               `yaml:"shm_size"`
	CPUQuota       int64               `yaml:"cpu_quota"`
	CPUShares      int64               `yaml:"cpu_shares"`
	CPUSet         string              `yaml:"cpuset"`
	OomKillDisable bool                `yaml:"oom_kill_disable"`

	AuthConfig struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Email    string `yaml:"email"`
		Token    string `yaml:"registry_token"`
	} `yaml:"auth_config"`

	Constraints Constraints `yaml:"when"`

	Vargs map[string]interface{} `yaml:",inline"`
}

// containerList is an intermediate type used for decoding a slice of containers
// in a format compatible with docker-compose.yml
type containerList struct {
	containers []*Container
}

// UnmarshalYAML implements custom Yaml unmarshaling.
func (c *containerList) UnmarshalYAML(unmarshal func(interface{}) error) error {
	slice := yaml.MapSlice{}
	err := unmarshal(&slice)
	if err != nil {
		return err
	}

	for _, s := range slice {
		cc := container{}

		out, merr := yaml.Marshal(s.Value)
		if err != nil {
			return merr
		}

		err = yaml.Unmarshal(out, &cc)
		if err != nil {
			return err
		}
		if cc.Name == "" {
			cc.Name = fmt.Sprintf("%v", s.Key)
		}
		if cc.Image == "" {
			cc.Image = fmt.Sprintf("%v", s.Key)
		}
		c.containers = append(c.containers, &Container{
			Name:           cc.Name,
			Image:          cc.Image,
			Build:          cc.Build,
			Pull:           cc.Pull,
			Detached:       cc.Detached,
			Privileged:     cc.Privileged,
			Environment:    cc.Environment.Map(),
			Labels:         cc.Labels.Map(),
			Entrypoint:     cc.Entrypoint.Slice(),
			Command:        cc.Command.Slice(),
			Commands:       cc.Commands.Slice(),
			ExtraHosts:     cc.ExtraHosts.Slice(),
			Volumes:        cc.Volumes.Slice(),
			VolumesFrom:    cc.VolumesFrom.Slice(),
			Devices:        cc.Devices.Slice(),
			Network:        cc.Network,
			DNS:            cc.DNS.Slice(),
			DNSSearch:      cc.DNSSearch.Slice(),
			MemSwapLimit:   cc.MemSwapLimit,
			MemLimit:       cc.MemLimit,
			ShmSize:        cc.ShmSize,
			CPUQuota:       cc.CPUQuota,
			CPUShares:      cc.CPUShares,
			CPUSet:         cc.CPUSet,
			OomKillDisable: cc.OomKillDisable,
			Vargs:          cc.Vargs,
			AuthConfig: Auth{
				Username: cc.AuthConfig.Username,
				Password: cc.AuthConfig.Password,
				Email:    cc.AuthConfig.Email,
			},
			Constraints: cc.Constraints,
		})
	}
	return err
}
