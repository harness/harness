package runner

import (
	"fmt"

	"github.com/drone/drone/engine/runner/parse"
)

// Spec defines the pipeline configuration and exeuction.
type Spec struct {
	// Volumes defines a list of all container volumes.
	Volumes []*Volume `json:"volumes,omitempty"`

	// Networks defines a list of all container networks.
	Networks []*Network `json:"networks,omitempty"`

	// Containers defines a list of all containers in the pipeline.
	Containers []*Container `json:"containers,omitempty"`

	// Nodes defines the container execution tree.
	Nodes *parse.Tree `json:"program,omitempty"`
}

// lookupContainer is a helper funciton that returns the named container from
// the slice of containers.
func (s *Spec) lookupContainer(name string) (*Container, error) {
	for _, container := range s.Containers {
		if container.Name == name {
			return container, nil
		}
	}
	return nil, fmt.Errorf("runner: unknown container %s", name)
}
