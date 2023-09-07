// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// Enums for event types delivered to the event stream for the UI
package enum

// EventType defines the kind of event
type EventType string

const (
	ExecutionUpdated   = "execution_updated"
	ExecutionRunning   = "execution_running"
	ExecutionCompleted = "execution_completed"
)
