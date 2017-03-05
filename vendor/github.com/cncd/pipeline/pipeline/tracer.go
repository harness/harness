package pipeline

import (
	"strconv"
	"time"
)

// Tracer handles process tracing.
type Tracer interface {
	Trace(*State) error
}

// TraceFunc type is an adapter to allow the use of ordinary
// functions as a Tracer.
type TraceFunc func(*State) error

// Trace calls f(proc, state).
func (f TraceFunc) Trace(state *State) error {
	return f(state)
}

// DefaultTracer provides a tracer that updates the CI_ enviornment
// variables to include the correct timestamp and status.
// TODO(bradrydzewski) find either a new home or better name for this.
var DefaultTracer = TraceFunc(func(state *State) error {
	if state.Process.Exited {
		return nil
	}
	if state.Pipeline.Step.Environment == nil {
		return nil
	}
	state.Pipeline.Step.Environment["CI_BUILD_STATUS"] = "success"
	state.Pipeline.Step.Environment["CI_BUILD_STARTED"] = strconv.FormatInt(state.Pipeline.Time, 10)
	state.Pipeline.Step.Environment["CI_BUILD_FINISHED"] = strconv.FormatInt(time.Now().Unix(), 10)

	state.Pipeline.Step.Environment["CI_JOB_STATUS"] = "success"
	state.Pipeline.Step.Environment["CI_JOB_STARTED"] = strconv.FormatInt(state.Pipeline.Time, 10)
	state.Pipeline.Step.Environment["CI_JOB_FINISHED"] = strconv.FormatInt(time.Now().Unix(), 10)

	if state.Pipeline.Error != nil {
		state.Pipeline.Step.Environment["CI_BUILD_STATUS"] = "failure"
		state.Pipeline.Step.Environment["CI_JOB_STATUS"] = "failure"
	}
	return nil
})
