package yaml

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

// Network defines a Docker network.
type Network struct {
	Name       string
	Driver     string
	DriverOpts map[string]string `yaml:"driver_opts"`
}

// networkList is an intermediate type used for decoding a slice of networks
// in a format compatible with docker-compose.yml
type networkList struct {
	networks []*Network
}

// UnmarshalYAML implements custom Yaml unmarshaling.
func (n *networkList) UnmarshalYAML(unmarshal func(interface{}) error) error {
	slice := yaml.MapSlice{}
	err := unmarshal(&slice)
	if err != nil {
		return err
	}

	for _, s := range slice {
		nn := Network{}

		out, merr := yaml.Marshal(s.Value)
		if merr != nil {
			return merr
		}

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
		n.networks = append(n.networks, &nn)
	}
	return err
}
