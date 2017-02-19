package yaml

import (
	"path/filepath"

	"github.com/drone/drone/yaml/types"
)

// Constraints define constraints for container execution.
type Constraints struct {
	Repo        Constraint
	Ref         Constraint
	Platform    Constraint
	Environment Constraint
	Event       Constraint
	Branch      Constraint
	Status      Constraint
	Matrix      ConstraintMap
	Local       types.BoolTrue
}

// Match returns true if all constraints match the given input. If a single constraint
// fails a false value is returned.
func (c *Constraints) Match(arch, target, event, branch, status string, matrix map[string]string) bool {
	return c.Platform.Match(arch) &&
		c.Environment.Match(target) &&
		c.Event.Match(event) &&
		c.Branch.Match(branch) &&
		c.Status.Match(status) &&
		c.Matrix.Match(matrix)
}

// Constraint defines an individual constraint.
type Constraint struct {
	Include []string
	Exclude []string
}

// Match returns true if the string matches the include patterns and does not
// match any of the exclude patterns.
func (c *Constraint) Match(v string) bool {
	if c.Excludes(v) {
		return false
	}
	if c.Includes(v) {
		return true
	}
	if len(c.Include) == 0 {
		return true
	}
	return false
}

// Includes returns true if the string matches matches the include patterns.
func (c *Constraint) Includes(v string) bool {
	for _, pattern := range c.Include {
		if ok, _ := filepath.Match(pattern, v); ok {
			return true
		}
	}
	return false
}

// Excludes returns true if the string matches matches the exclude patterns.
func (c *Constraint) Excludes(v string) bool {
	for _, pattern := range c.Exclude {
		if ok, _ := filepath.Match(pattern, v); ok {
			return true
		}
	}
	return false
}

// UnmarshalYAML implements custom Yaml unmarshaling.
func (c *Constraint) UnmarshalYAML(unmarshal func(interface{}) error) error {

	var out1 = struct {
		Include types.StringOrSlice
		Exclude types.StringOrSlice
	}{}

	var out2 types.StringOrSlice

	unmarshal(&out1)
	unmarshal(&out2)

	c.Exclude = out1.Exclude.Slice()
	c.Include = append(
		out1.Include.Slice(),
		out2.Slice()...,
	)
	return nil
}

// ConstraintMap defines an individual constraint for key value structures.
type ConstraintMap struct {
	Include map[string]string
	Exclude map[string]string
}

// Match returns true if the params matches the include key values and does not
// match any of the exclude key values.
func (c *ConstraintMap) Match(params map[string]string) bool {
	// when no includes or excludes automatically match
	if len(c.Include) == 0 && len(c.Exclude) == 0 {
		return true
	}

	// exclusions are processed first. So we can include everything and then
	// selectively include others.
	if len(c.Exclude) != 0 {
		var matches int

		for key, val := range c.Exclude {
			if params[key] == val {
				matches++
			}
		}
		if matches == len(c.Exclude) {
			return false
		}
	}

	for key, val := range c.Include {
		if params[key] != val {
			return false
		}
	}

	return true
}

// UnmarshalYAML implements custom Yaml unmarshaling.
func (c *ConstraintMap) UnmarshalYAML(unmarshal func(interface{}) error) error {

	out1 := struct {
		Include map[string]string
		Exclude map[string]string
	}{
		Include: map[string]string{},
		Exclude: map[string]string{},
	}

	out2 := map[string]string{}

	unmarshal(&out1)
	unmarshal(&out2)

	c.Include = out1.Include
	c.Exclude = out1.Exclude
	for k, v := range out2 {
		c.Include[k] = v
	}
	return nil
}
