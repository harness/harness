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

package hook

// Output represents the output of server hook api calls.
type Output struct {
	// Messages contains standard user facing messages.
	Messages []string `json:"messages,omitempty"`

	// Error contains the user facing error (like "branch is protected", ...).
	Error *string `json:"error,omitempty"`
}

// ReferenceUpdate represents an update of a git reference.
type ReferenceUpdate struct {
	// Ref is the full name of the reference that got updated.
	Ref string `json:"ref"`
	// Old is the old commmit hash (before the update).
	Old string `json:"old"`
	// New is the new commit hash (after the update).
	New string `json:"new"`
}

// PostReceiveInput represents the input of the post-receive git hook.
type PostReceiveInput struct {
	// RefUpdates contains all references that got updated as part of the git operation.
	RefUpdates []ReferenceUpdate `json:"ref_updates"`
}

// PreReceiveInput represents the input of the pre-receive git hook.
type PreReceiveInput struct {
	// RefUpdates contains all references that are being updated as part of the git operation.
	RefUpdates []ReferenceUpdate `json:"ref_updates"`
}

// UpdateInput represents the input of the update git hook.
type UpdateInput struct {
	// RefUpdate contains information about the reference that is being updated.
	RefUpdate ReferenceUpdate `json:"ref_update"`
}
