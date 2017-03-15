package backend

import "io"

// Engine defines a container orchestration backend and is used
// to create and manage container resources.
type Engine interface {
	// Setup the pipeline environment.
	Setup(*Config) error
	// Start the pipeline step.
	Exec(*Step) error
	// Kill the pipeline step.
	Kill(*Step) error
	// Wait for the pipeline step to complete and returns
	// the completion results.
	Wait(*Step) (*State, error)
	// Tail the pipeline step logs.
	Tail(*Step) (io.ReadCloser, error)
	// Destroy the pipeline environment.
	Destroy(*Config) error
}
