// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// Enums for event types delivered to the event stream for the UI
package enum

// SSEType defines the kind of server sent event
type SSEType string

const (
	SSETypeExecutionUpdated   = "execution_updated"
	SSETypeExecutionRunning   = "execution_running"
	SSETypeExecutionCompleted = "execution_completed"
	SSETypeExecutionCanceled  = "execution_canceled"

	SSETypeRepositoryImportCompleted = "repository_import_completed"
	SSETypeRepositoryExportCompleted = "repository_export_completed"
)
