// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package core

import "context"

// Status types.
const (
	StatusSkipped  = "skipped"
	StatusBlocked  = "blocked"
	StatusDeclined = "declined"
	StatusWaiting  = "waiting_on_dependencies"
	StatusPending  = "pending"
	StatusRunning  = "running"
	StatusPassing  = "success"
	StatusFailing  = "failure"
	StatusKilled   = "killed"
	StatusError    = "error"
)

type (
	// Status represents a commit status.
	Status struct {
		State  string
		Label  string
		Desc   string
		Target string
	}

	// StatusInput provides the necessary metadata to
	// set the commit or deployment status.
	StatusInput struct {
		Repo  *Repository
		Build *Build
	}

	// StatusService sends the commit status to an external
	// external source code management service (e.g. GitHub).
	StatusService interface {
		Send(ctx context.Context, user *User, req *StatusInput) error
	}
)
