package yaml

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

type (
	// Networks defines a collection of networks.
	Networks struct {
		Networks []*Network
	}

	// Network defines a container network.
	Network struct {
		Name       string            `yaml:"name,omitempty"`
		Driver     string            `yaml:"driver,omitempty"`
		DriverOpts map[string]string `yaml:"driver_opts,omitempty"`
	}
)

// UnmarshalYAML implements the Unmarshaller interface.
func (n *Networks) UnmarshalYAML(unmarshal func(interface{}) error) error {
	slice := yaml.MapSlice{}
	err := unmarshal(&slice)
	if err != nil {
		return err
	}

	for _, s := range slice {
		nn := Network{}
		out, _ := yaml.Marshal(s.Value)

		err = yaml.Unmarshal(out, &nn)
		if err != nil {
			return err
		}
		if nn.Name == "" {
			nn.Name = fmt.Sprintf("%v", s.Key)
		}
		if nn.Driver == "" {
			nn.Driver = "bridge"
		}
		n.Networks = append(n.Networks, &nn)
	}
	return err
}
