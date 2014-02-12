package scm

import (
	"strconv"
)

const (
	DefaultGitDepth = 50
)

// Scm stores the scm tool configurations.
type Scm struct {
	Git *Git `yaml:"git,omitempty"`
}

// Git stores the configuration details for
// executing Git commands.
type Git struct {
	Depth string `yaml:"depth,omitempty"`
}

// GitDepth returns scm.Git.Depth
// when it's not empty.
// GitDepth returns GitDefaultDepth
// when scm.Git.Depth is empty.
func GitDepth(scm *Scm) int {
	if scm == nil || scm.Git == nil || scm.Git.Depth == "" {
		return DefaultGitDepth
	}
	d, err := strconv.Atoi(scm.Git.Depth)
	if err != nil {
		return DefaultGitDepth
	}
	return d
}
