package parse

import "strings"

// mapEqualSlice represents a map[string]string or a slice of
// strings in key=value format.
type mapEqualSlice struct {
	parts map[string]string
}

func (s *mapEqualSlice) UnmarshalYAML(unmarshal func(interface{}) error) error {
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

// stringOrSlice represents a string or an array of strings.
type stringOrSlice struct {
	parts []string
}

func (s *stringOrSlice) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var sliceType []string
	err := unmarshal(&sliceType)
	if err == nil {
		s.parts = sliceType
		return nil
	}

	var stringType string
	err = unmarshal(&stringType)
	if err == nil {
		sliceType = make([]string, 0, 1)
		s.parts = append(sliceType, string(stringType))
		return nil
	}
	return err
}
