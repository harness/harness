// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package job

import (
	"errors"
	"time"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type Definition struct {
	UID        string
	Type       string
	MaxRetries int
	Timeout    time.Duration
	Data       string
}

func (def *Definition) Validate() error {
	if def.Type == "" {
		return errors.New("job Type must not be empty")
	}

	if def.UID == "" {
		return errors.New("job must have unique identifier")
	}

	if def.MaxRetries < 0 {
		return errors.New("job MaxRetries must be positive")
	}

	if def.Timeout < time.Second {
		return errors.New("job Timeout too short")
	}

	return nil
}

func (def *Definition) toNewJob() *types.Job {
	nowMilli := time.Now().UnixMilli()
	return &types.Job{
		UID:                 def.UID,
		Created:             nowMilli,
		Updated:             nowMilli,
		Type:                def.Type,
		Priority:            enum.JobPriorityNormal,
		Data:                def.Data,
		Result:              "",
		MaxDurationSeconds:  int(def.Timeout / time.Second),
		MaxRetries:          def.MaxRetries,
		State:               enum.JobStateScheduled,
		Scheduled:           nowMilli,
		TotalExecutions:     0,
		RunBy:               "",
		RunDeadline:         nowMilli,
		RunProgress:         ProgressMin,
		LastExecuted:        0, // never executed
		IsRecurring:         false,
		RecurringCron:       "",
		ConsecutiveFailures: 0,
		LastFailureError:    "",
	}
}
