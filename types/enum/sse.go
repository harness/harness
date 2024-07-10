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

package enum

// SSEType defines the kind of server sent event.
type SSEType string

// Enums for event types delivered to the event stream for the UI.
const (
	SSETypeExecutionUpdated   SSEType = "execution_updated"
	SSETypeExecutionRunning   SSEType = "execution_running"
	SSETypeExecutionCompleted SSEType = "execution_completed"
	SSETypeExecutionCanceled  SSEType = "execution_canceled"

	SSETypeRepositoryImportCompleted SSEType = "repository_import_completed"
	SSETypeRepositoryExportCompleted SSEType = "repository_export_completed"

	SSETypePullRequestUpdated SSEType = "pullreq_updated"

	SSETypeLogLineAppended SSEType = "log_line_appended"
)
