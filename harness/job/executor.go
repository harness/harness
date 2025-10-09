// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package job

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/harness/gitness/pubsub"

	"github.com/rs/zerolog/log"
)

// Executor holds map of Handler objects per each job type registered.
// The Scheduler uses the Executor to start execution of jobs.
type Executor struct {
	handlerMap      map[string]Handler
	handlerComplete bool
	store           Store
	publisher       pubsub.Publisher
}

const (
	ProgressMin = 0
	ProgressMax = 100
)

// ProgressReporter can be used by a job Handler to report back the execution progress.
type ProgressReporter func(progress int, result string) error

// Handler is a job executor for a specific job type.
// An implementation should try to honor the context and
// try to abort the execution as soon as the context is done.
type Handler interface {
	Handle(ctx context.Context, input string, fn ProgressReporter) (result string, err error)
}

var errNoHandlerDefined = errors.New("no handler registered for the job type")

// NewExecutor creates new Executor.
func NewExecutor(store Store, publisher pubsub.Publisher) *Executor {
	return &Executor{
		handlerMap:      make(map[string]Handler),
		handlerComplete: false,
		store:           store,
		publisher:       publisher,
	}
}

// Register registers a job Handler for the provided job type.
// This function is not thread safe. All calls are expected to be made
// in a single thread during the application boot time.
func (e *Executor) Register(jobType string, exec Handler) error {
	if jobType == "" {
		return errors.New("jobType must not be empty")
	}

	if e.handlerComplete {
		return errors.New("job handler registration is complete")
	}

	if exec == nil {
		return errors.New("provided Handler is nil")
	}

	if _, ok := e.handlerMap[jobType]; ok {
		return fmt.Errorf("a Handler is already defined to run the '%s' job types", jobType)
	}

	e.handlerMap[jobType] = exec

	return nil
}

// finishRegistration forbids further registration of job types.
// It is called by the Scheduler when it starts.
func (e *Executor) finishRegistration() {
	e.handlerComplete = true
}

// exec runs a single job. This function is synchronous,
// so the caller is responsible to run it in a separate go-routine.
func (e *Executor) exec(
	ctx context.Context,
	jobUID, jobType string,
	input string,
) (result string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf(
				"panic while processing job=%s type=%s: %v\n%s",
				jobUID, jobType, r, debug.Stack())
		}
	}()

	exec, ok := e.handlerMap[jobType]
	if !ok {
		return "", errNoHandlerDefined
	}

	// progressReporter is the function with which the job can update its progress.
	// This function will be executed in the job executor's Go-routine.
	// It uses the job's context.
	progressReporter := func(progress int, result string) error {
		if progress < ProgressMin || progress > ProgressMax {
			return errors.New("progress must be between 0 and 100")
		}

		jobDummy := &Job{
			UID:         jobUID,
			Type:        jobType,
			Updated:     time.Now().UnixMilli(),
			Result:      result,
			State:       JobStateRunning,
			RunProgress: progress,
		}

		// This doesn't need to be behind the global lock because it only updates the single row.
		// While a job is running no other process should touch it.
		// Even this call will fail if the context deadline has been exceeded.
		// The job parameter is a dummy Job object that just holds fields that should be updated.
		if err := e.store.UpdateProgress(ctx, jobDummy); err != nil {
			return err
		}

		// tell everybody that a job progress has been updated
		if err := publishStateChange(ctx, e.publisher, jobDummy); err != nil {
			log.Err(err).Msg("failed to publish job state change")
		}

		return nil
	}

	return exec.Handle(ctx, input, progressReporter) // runs the job
}

func FailProgress() Progress {
	return Progress{
		State:    JobStateFailed,
		Progress: ProgressMax,
		Result:   "",
		Failure:  "",
	}
}
