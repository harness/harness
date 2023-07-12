// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package githook

// Output represents the output of server hook api calls.
// TODO: support non-error messages (once we need it).
type Output struct {
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
