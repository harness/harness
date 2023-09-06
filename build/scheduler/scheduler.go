// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package scheduler

import (
	"context"

	"github.com/harness/gitness/types"
)

// Filter provides filter criteria to limit stages requested
// from the scheduler.
type Filter struct {
	Kind    string
	Type    string
	OS      string
	Arch    string
	Kernel  string
	Variant string
	Labels  map[string]string
}

// Scheduler schedules Build stages for execution.
type Scheduler interface {
	// Schedule schedules the stage for execution.
	Schedule(ctx context.Context, stage *types.Stage) error

	// Request requests the next stage scheduled for execution.
	Request(ctx context.Context, filter Filter) (*types.Stage, error)
}
