package parse

import (
	"fmt"

	"github.com/drone/drone/engine/runner"

	"gopkg.in/yaml.v2"
)

type Conditions struct {
	Platform    []string
	Environment []string
	Event       []string
	Branch      []string
	Status      []string
	Matrix      map[string]string
}

// ContainerNode represents a Docker container.
type ContainerNode struct {
	NodeType

	// Container represents the container configuration.
	Container  runner.Container
	Conditions Conditions
	Disabled   bool
	Commands   []string
	Vargs      map[string]interface{}

	root *RootNode
}

// Root returns the root node.
func (n *ContainerNode) Root() *RootNode { return n.root }

// OnSuccess returns true if the container should be executed
// when the exit code of the previous step is 0.
func (n *ContainerNode) OnSuccess() bool {
	for _, status := range n.Conditions.Status {
		if status == "success" {
			return true
		}
	}
	return false
}

// OnFailure returns true if the container should be executed
// even when the exit code of the previous step != 0.
func (n *ContainerNode) OnFailure() bool {
	for _, status := range n.Conditions.Status {
		if status == "failure" {
			return true
		}
	}
	return false
}

//
// intermediate types for yaml decoding.
//

type container struct {
	Name           string        `yaml:"name"`
	Image          string        `yaml:"image"`
	Build          string        `yaml:"build"`
	Pull           bool          `yaml:"pull"`
	Privileged     bool          `yaml:"privileged"`
	Environment    mapEqualSlice `yaml:"environment"`
	Entrypoint     stringOrSlice `yaml:"entrypoint"`
	Command        stringOrSlice `yaml:"command"`
	Commands       stringOrSlice `yaml:"commands"`
	ExtraHosts     stringOrSlice `yaml:"extra_hosts"`
	Volumes        stringOrSlice `yaml:"volumes"`
	VolumesFrom    stringOrSlice `yaml:"volumes_from"`
	Devices        stringOrSlice `yaml:"devices"`
	Network        string        `yaml:"network_mode"`
	DNS            stringOrSlice `yaml:"dns"`
	DNSSearch      stringOrSlice `yaml:"dns_search"`
	MemSwapLimit   int64         `yaml:"memswap_limit"`
	MemLimit       int64         `yaml:"mem_limit"`
	CPUQuota       int64         `yaml:"cpu_quota"`
	CPUShares      int64         `yaml:"cpu_shares"`
	CPUSet         string        `yaml:"cpuset"`
	OomKillDisable bool          `yaml:"oom_kill_disable"`

	AuthConfig struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Email    string `yaml:"email"`
		Token    string `yaml:"registry_token"`
	} `yaml:"auth_config"`

	Conditions struct {
		Platform    stringOrSlice     `yaml:"platform"`
		Environment stringOrSlice     `yaml:"environment"`
		Event       stringOrSlice     `yaml:"event"`
		Branch      stringOrSlice     `yaml:"branch"`
		Status      stringOrSlice     `yaml:"status"`
		Matrix      map[string]string `yaml:"matrix"`
	} `yaml:"when"`

	Vargs map[string]interface{} `yaml:",inline"`
}

func (c *container) ToContainer() runner.Container {
	return runner.Container{
		Name:           c.Name,
		Image:          c.Image,
		Pull:           c.Pull,
		Privileged:     c.Privileged,
		Environment:    c.Environment.parts,
		Entrypoint:     c.Entrypoint.parts,
		Command:        c.Command.parts,
		ExtraHosts:     c.ExtraHosts.parts,
		Volumes:        c.Volumes.parts,
		VolumesFrom:    c.VolumesFrom.parts,
		Devices:        c.Devices.parts,
		Network:        c.Network,
		DNS:            c.DNS.parts,
		DNSSearch:      c.DNSSearch.parts,
		MemSwapLimit:   c.MemSwapLimit,
		MemLimit:       c.MemLimit,
		CPUQuota:       c.CPUQuota,
		CPUShares:      c.CPUShares,
		CPUSet:         c.CPUSet,
		OomKillDisable: c.OomKillDisable,
		AuthConfig: runner.Auth{
			Username: c.AuthConfig.Username,
			Password: c.AuthConfig.Password,
			Email:    c.AuthConfig.Email,
			Token:    c.AuthConfig.Token,
		},
	}
}

func (c *container) ToConditions() Conditions {
	return Conditions{
		Platform:    c.Conditions.Platform.parts,
		Environment: c.Conditions.Environment.parts,
		Event:       c.Conditions.Event.parts,
		Branch:      c.Conditions.Branch.parts,
		Status:      c.Conditions.Status.parts,
		Matrix:      c.Conditions.Matrix,
	}
}

type containerList struct {
	containers []*container
}

func (c *containerList) UnmarshalYAML(unmarshal func(interface{}) error) error {
	slice := yaml.MapSlice{}
	err := unmarshal(&slice)
	if err != nil {
		return err
	}

	for _, s := range slice {
		cc := container{}

		out, err := yaml.Marshal(s.Value)
		if err != nil {
			return err
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
		c.containers = append(c.containers, &cc)
	}
	return err
}
