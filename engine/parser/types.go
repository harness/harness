package parser

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

func (s stringOrSlice) Slice() []string {
	return s.parts
}
