// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package webhook

// RepoMetadata describes the repo related metadata of webhook payload.
// TODO: move in separate package for small import?
type RepoMetadata struct {
	ID            int64  `json:"id"`
	Path          string `json:"path"`
	UID           string `json:"uid"`
	DefaultBranch string `json:"default_branch"`
	GitURL        string `json:"git_url,omitempty"` // TODO: remove once handled properly.
}
