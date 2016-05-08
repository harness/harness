package yaml

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

// Volume defines a Docker volume.
type Volume struct {
	Name       string
	Driver     string
	DriverOpts map[string]string `yaml:"driver_opts"`
	External   bool
}

// volumeList is an intermediate type used for decoding a slice of volumes
// in a format compatible with docker-compose.yml
type volumeList struct {
	volumes []*Volume
}

// UnmarshalYAML implements custom Yaml unmarshaling.
func (v *volumeList) UnmarshalYAML(unmarshal func(interface{}) error) error {
	slice := yaml.MapSlice{}
	err := unmarshal(&slice)
	if err != nil {
		return err
	}

	for _, s := range slice {
		vv := Volume{}
		out, merr := yaml.Marshal(s.Value)
		if merr != nil {
			return merr
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
