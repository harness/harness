package common

import (
	"path/filepath"
	"strings"
)

// Config represents a repository build configuration.
type Config struct {
	Setup *Step
	Clone *Step
	Build *Step

	Compose map[string]*Step
	Publish map[string]*Step
	Deploy  map[string]*Step
	Notify  map[string]*Step

	Matrix Matrix
	Axis   Axis
}

// Matrix represents the build matrix.
type Matrix map[string][]string

// Axis represents a single permutation of entries
// from the build matrix.
type Axis map[string]string

// String returns a string representation of an Axis as
// a comma-separated list of environment variables.
func (a Axis) String() string {
	var envs []string
	for k, v := range a {
		envs = append(envs, k+"="+v)
	}
	return strings.Join(envs, " ")
}

// Step represents a step in the build process, including
// the execution environment and parameters.
type Step struct {
	Image       string
	Pull        bool
	Privileged  bool
	Environment []string
	Entrypoint  []string
	Command     []string
	Volumes     []string
	WorkingDir  string `yaml:"working_dir"`
	NetworkMode string `yaml:"net"`

	// Condition represents a set of conditions that must
	// be met in order to execute this step.
	Condition *Condition `yaml:"when"`

	// Config represents the unique configuration details
	// for each plugin.
	Config map[string]interface{} `yaml:"config,inline"`
}

// Condition represents a set of conditions that must
// be met in order to proceed with a build or build step.
type Condition struct {
	Owner  string // Indicates the step should run only for this repo (useful for forks)
	Branch string // Indicates the step should run only for this branch

	// Indicates the step should only run when the following
	// matrix values are present for the sub-build.
	Matrix map[string]string
}

// MatchBranch is a helper function that returns true
// if all_branches is true. Else it returns false if a
// branch condition is specified, and the branch does
// not match.
func (c *Condition) MatchBranch(branch string) bool {
	if len(c.Branch) == 0 {
		return true
	}
	match, _ := filepath.Match(c.Branch, branch)
	return match
}

// MatchOwner is a helper function that returns false
// if an owner condition is specified and the repository
// owner does not match.
//
// This is useful when you want to prevent forks from
// executing deployment, publish or notification steps.
func (c *Condition) MatchOwner(owner string) bool {
	if len(c.Owner) == 0 {
		return true
	}
	parts := strings.Split(owner, "/")
	switch len(parts) {
	case 2:
		return c.Owner == parts[0]
	case 3:
		return c.Owner == parts[1]
	default:
		return c.Owner == owner
	}
}
