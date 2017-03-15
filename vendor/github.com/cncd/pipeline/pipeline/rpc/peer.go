package rpc

import (
	"context"
	"io"

	"github.com/cncd/pipeline/pipeline/backend"
)

// ErrCancelled signals the pipeine is cancelled.
// var ErrCancelled = errors.New("cancelled")

type (
	// Filter defines filters for fetching items from the queue.
	Filter struct {
		Labels map[string]string `json:"labels"`
		Expr   string            `json:"expr"`
	}

	// State defines the pipeline state.
	State struct {
		Proc     string `json:"proc"`
		Exited   bool   `json:"exited"`
		ExitCode int    `json:"exit_code"`
		Started  int64  `json:"started"`
		Finished int64  `json:"finished"`
		Error    string `json:"error"`
	}

	// Pipeline defines the pipeline execution details.
	Pipeline struct {
		ID      string          `json:"id"`
		Config  *backend.Config `json:"config"`
		Timeout int64           `json:"timeout"`
	}
)

// NoFilter is an empty filter.
var NoFilter = Filter{}

// Peer defines a peer-to-peer connection.
type Peer interface {
	// Next returns the next pipeline in the queue.
	Next(c context.Context, f Filter) (*Pipeline, error)

	// Wait blocks untilthe pipeline is complete.
	Wait(c context.Context, id string) error

	// Done signals the pipeline is complete.
	Done(c context.Context, id string) error

	// Extend extends the pipeline deadline
	Extend(c context.Context, id string) error

	// Update updates the pipeline state.
	Update(c context.Context, id string, state State) error

	// Upload uploads the pipeline artifact.
	Upload(c context.Context, id, mime string, file io.Reader) error

	// Log writes the pipeline log entry.
	Log(c context.Context, id string, line *Line) error
}
