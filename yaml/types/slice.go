package types

// StringOrSlice is a custom Yaml type that can hold a string or slice of strings.
type StringOrSlice struct {
	parts []string
}

// UnmarshalYAML implements custom Yaml unmarshaling.
func (s *StringOrSlice) UnmarshalYAML(unmarshal func(interface{}) error) error {
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

// Slice returns the slice of strings.
func (s StringOrSlice) Slice() []string {
	return s.parts
}

// NewStringOrSlice returns a new StringOrSlice.
func NewStringOrSlice(from []string) *StringOrSlice {
	return &StringOrSlice{from}
}
