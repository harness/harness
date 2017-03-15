package yaml

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

type (
	// Volumes defines a collection of volumes.
	Volumes struct {
		Volumes []*Volume
	}

	// Volume defines a container volume.
	Volume struct {
		Name       string            `yaml:"name,omitempty"`
		Driver     string            `yaml:"driver,omitempty"`
		DriverOpts map[string]string `yaml:"driver_opts,omitempty"`
	}
)

// UnmarshalYAML implements the Unmarshaller interface.
func (v *Volumes) UnmarshalYAML(unmarshal func(interface{}) error) error {
	slice := yaml.MapSlice{}
	err := unmarshal(&slice)
	if err != nil {
		return err
	}

	for _, s := range slice {
		vv := Volume{}
		out, _ := yaml.Marshal(s.Value)

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
		v.Volumes = append(v.Volumes, &vv)
	}
	return err
}
