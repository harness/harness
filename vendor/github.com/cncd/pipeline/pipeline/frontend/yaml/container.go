package yaml

import (
	"fmt"

	libcompose "github.com/docker/libcompose/yaml"
	"gopkg.in/yaml.v2"
)

type (
	// AuthConfig defines registry authentication credentials.
	AuthConfig struct {
		Username string
		Password string
		Email    string
	}

	// Containers denotes an ordered collection of containers.
	Containers struct {
		Containers []*Container
	}

	// Container defines a container.
	Container struct {
		AuthConfig    AuthConfig                `yaml:"auth_config,omitempty"`
		CapAdd        []string                  `yaml:"cap_add,omitempty"`
		CapDrop       []string                  `yaml:"cap_drop,omitempty"`
		Command       libcompose.Command        `yaml:"command,omitempty"`
		Commands      libcompose.Stringorslice  `yaml:"commands,omitempty"`
		CPUQuota      libcompose.StringorInt    `yaml:"cpu_quota,omitempty"`
		CPUSet        string                    `yaml:"cpuset,omitempty"`
		CPUShares     libcompose.StringorInt    `yaml:"cpu_shares,omitempty"`
		Detached      bool                      `yaml:"detach,omitempty"`
		Devices       []string                  `yaml:"devices,omitempty"`
		DNS           libcompose.Stringorslice  `yaml:"dns,omitempty"`
		DNSSearch     libcompose.Stringorslice  `yaml:"dns_search,omitempty"`
		Entrypoint    libcompose.Command        `yaml:"entrypoint,omitempty"`
		Environment   libcompose.SliceorMap     `yaml:"environment,omitempty"`
		ExtraHosts    []string                  `yaml:"extra_hosts,omitempty"`
		Group         string                    `yaml:"group,omitempty"`
		Image         string                    `yaml:"image,omitempty"`
		Isolation     string                    `yaml:"isolation,omitempty"`
		Labels        libcompose.SliceorMap     `yaml:"labels,omitempty"`
		MemLimit      libcompose.MemStringorInt `yaml:"mem_limit,omitempty"`
		MemSwapLimit  libcompose.MemStringorInt `yaml:"memswap_limit,omitempty"`
		MemSwappiness libcompose.MemStringorInt `yaml:"mem_swappiness,omitempty"`
		Name          string                    `yaml:"name,omitempty"`
		NetworkMode   string                    `yaml:"network_mode,omitempty"`
		Networks      libcompose.Networks       `yaml:"networks,omitempty"`
		Privileged    bool                      `yaml:"privileged,omitempty"`
		Pull          bool                      `yaml:"pull,omitempty"`
		ShmSize       libcompose.MemStringorInt `yaml:"shm_size,omitempty"`
		Ulimits       libcompose.Ulimits        `yaml:"ulimits,omitempty"`
		Volumes       libcompose.Volumes        `yaml:"volumes,omitempty"`
		Secrets       Secrets                   `yaml:"secrets,omitempty"`
		Constraints   Constraints               `yaml:"when,omitempty"`
		Vargs         map[string]interface{}    `yaml:",inline"`
	}
)

// UnmarshalYAML implements the Unmarshaller interface.
func (c *Containers) UnmarshalYAML(unmarshal func(interface{}) error) error {
	slice := yaml.MapSlice{}
	if err := unmarshal(&slice); err != nil {
		return err
	}

	for _, s := range slice {
		container := Container{}
		out, _ := yaml.Marshal(s.Value)

		if err := yaml.Unmarshal(out, &container); err != nil {
			return err
		}
		if container.Name == "" {
			container.Name = fmt.Sprintf("%v", s.Key)
		}
		c.Containers = append(c.Containers, &container)
	}
	return nil
}
