package parse

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

// NetworkNode represents a Docker network.
type NetworkNode struct {
	NodeType
	root *RootNode

	Name       string
	Driver     string
	DriverOpts map[string]string
}

// Root returns the root node.
func (n *NetworkNode) Root() *RootNode { return n.root }

//
// intermediate types for yaml decoding.
//

// network is an intermediate type used for decoding a networks in a format
// compatible with docker-compose.yml
type network struct {
	Name       string
	Driver     string
	DriverOpts map[string]string `yaml:"driver_opts"`
}

// networkList is an intermediate type used for decoding a slice of networks
// in a format compatible with docker-compose.yml
type networkList struct {
	networks []*network
}

func (n *networkList) UnmarshalYAML(unmarshal func(interface{}) error) error {
	slice := yaml.MapSlice{}
	err := unmarshal(&slice)
	if err != nil {
		return err
	}

	for _, s := range slice {
		nn := network{}

		out, err := yaml.Marshal(s.Value)
		if err != nil {
			return err
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
