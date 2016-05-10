package types

import "strings"

// MapEqualSlice is a custom Yaml type that can hold a map or slice of strings
// in key=value format.
type MapEqualSlice struct {
	parts map[string]string
}

// UnmarshalYAML implements custom Yaml unmarshaling.
func (s *MapEqualSlice) UnmarshalYAML(unmarshal func(interface{}) error) error {
	s.parts = map[string]string{}
	err := unmarshal(&s.parts)
	if err == nil {
		return nil
	}

	var slice []string
	err = unmarshal(&slice)
	if err != nil {
		return err
	}
	for _, v := range slice {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			val := parts[1]
			s.parts[key] = val
		}
	}
	return nil
}

// Map returns the Yaml information as a map.
func (s *MapEqualSlice) Map() map[string]string {
	return s.parts
}

// NewMapEqualSlice returns a new MapEqualSlice.
func NewMapEqualSlice(from map[string]string) *MapEqualSlice {
	return &MapEqualSlice{from}
}
