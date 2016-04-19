package runner

//go:generate mockery -name Engine -output mock -case=underscore

import "io"

// Engine defines the container runtime engine.
type Engine interface {
	// VolumeCreate(*Volume) (string, error)
	// VolumeRemove(string) error
	ContainerStart(*Container) (string, error)
	ContainerStop(string) error
	ContainerRemove(string) error
	ContainerWait(string) (*State, error)
	ContainerLogs(string) (io.ReadCloser, error)
}

// State defines the state of the container.
type State struct {
	ExitCode  int  // container exit code
	OOMKilled bool // container exited due to oom error
}
