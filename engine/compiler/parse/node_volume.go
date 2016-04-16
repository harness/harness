package parse

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

// VolumeNode represents a Docker volume.
type VolumeNode struct {
	NodeType
	root *RootNode

	Name       string
	Driver     string
	DriverOpts map[string]string
	External   bool
}

// Root returns the root node.
func (n *VolumeNode) Root() *RootNode { return n.root }

//
// intermediate types for yaml decoding.
//

// volume is an intermediate type used for decoding a volumes in a format
// compatible with docker-compose.yml
type volume struct {
	Name       string
	Driver     string
	DriverOpts map[string]string `yaml:"driver_opts"`
}

// volumeList is an intermediate type used for decoding a slice of volumes
// in a format compatible with docker-compose.yml
type volumeList struct {
	volumes []*volume
}

func (v *volumeList) UnmarshalYAML(unmarshal func(interface{}) error) error {
	slice := yaml.MapSlice{}
	err := unmarshal(&slice)
	if err != nil {
		return err
	}

	for _, s := range slice {
		vv := volume{}

		out, err := yaml.Marshal(s.Value)
		if err != nil {
			return err
		}

		err = yaml.Unmarshal(out, &vv)
		if err != nil {
			return err
		}
		if vv.Name == "" {
			vv.Name = fmt.Sprintf("%v", s.Key)
		}
		if vv.Driver == "" {
			vv.Driver = "local"
		}
		v.volumes = append(v.volumes, &vv)
	}
	return err
}
