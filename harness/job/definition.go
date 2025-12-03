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
	"errors"
	"time"
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

func (def *Definition) toNewJob() *Job {
	nowMilli := time.Now().UnixMilli()
	return &Job{
		UID:                 def.UID,
		Created:             nowMilli,
		Updated:             nowMilli,
		Type:                def.Type,
		Priority:            JobPriorityNormal,
		Data:                def.Data,
		Result:              "",
		MaxDurationSeconds:  int(def.Timeout / time.Second),
		MaxRetries:          def.MaxRetries,
		State:               JobStateScheduled,
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
