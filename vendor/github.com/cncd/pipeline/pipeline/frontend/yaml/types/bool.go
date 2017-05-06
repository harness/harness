package types

import "strconv"

// BoolTrue is a custom Yaml boolean type that defaults to true.
type BoolTrue struct {
	value bool
}

// UnmarshalYAML implements custom Yaml unmarshaling.
func (b *BoolTrue) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	err := unmarshal(&s)
	if err != nil {
		return err
	}

	value, err := strconv.ParseBool(s)
	if err == nil {
		b.value = !value
	}
	return nil
}

// Bool returns the bool value.
func (b BoolTrue) Bool() bool {
	return !b.value
}