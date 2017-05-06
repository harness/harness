package yaml

import (
	"path/filepath"

	"github.com/cncd/pipeline/pipeline/frontend"
	libcompose "github.com/docker/libcompose/yaml"
	"github.com/cncd/pipeline/pipeline/frontend/yaml/types"
)

type (
	// Constraints defines a set of runtime constraints.
	Constraints struct {
		Repo        Constraint
		Instance    Constraint
		Platform    Constraint
		Environment Constraint
		Event       Constraint
		Branch      Constraint
		Status      Constraint
		Matrix      ConstraintMap
		Local       types.BoolTrue
	}

	// Constraint defines a runtime constraint.
	Constraint struct {
		Include []string
		Exclude []string
	}

	// ConstraintMap defines a runtime constraint map.
	ConstraintMap struct {
		Include map[string]string
		Exclude map[string]string
	}
)

// Match returns true if all constraints match the given input. If a single
// constraint fails a false value is returned.
func (c *Constraints) Match(metadata frontend.Metadata) bool {
	return c.Platform.Match(metadata.Sys.Arch) &&
		c.Environment.Match(metadata.Curr.Target) &&
		c.Event.Match(metadata.Curr.Event) &&
		c.Branch.Match(metadata.Curr.Commit.Branch) &&
		c.Repo.Match(metadata.Repo.Name) &&
		c.Matrix.Match(metadata.Job.Matrix)
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

// Includes returns true if the string matches the include patterns.
func (c *Constraint) Includes(v string) bool {
	for _, pattern := range c.Include {
		if ok, _ := filepath.Match(pattern, v); ok {
			return true
		}
	}
	return false
}

// Excludes returns true if the string matches the exclude patterns.
func (c *Constraint) Excludes(v string) bool {
	for _, pattern := range c.Exclude {
		if ok, _ := filepath.Match(pattern, v); ok {
			return true
		}
	}
	return false
}

// UnmarshalYAML unmarshals the constraint.
func (c *Constraint) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var out1 = struct {
		Include libcompose.Stringorslice
		Exclude libcompose.Stringorslice
	}{}

	var out2 libcompose.Stringorslice

	unmarshal(&out1)
	unmarshal(&out2)

	c.Exclude = out1.Exclude
	c.Include = append(
		out1.Include,
		out2...,
	)
	return nil
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

// UnmarshalYAML unmarshals the constraint map.
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
