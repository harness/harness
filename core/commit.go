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

type (
	// Commit represents a git commit.
	Commit struct {
		Sha       string
		Ref       string
		Message   string
		Author    *Committer
		Committer *Committer
		Link      string
	}

	// Committer represents the commit author.
	Committer struct {
		Name   string
		Email  string
		Date   int64
		Login  string
		Avatar string
	}

	// Change represents a file change in a commit.
	Change struct {
		Path    string
		Added   bool
		Renamed bool
		Deleted bool
	}

	// CommitService provides access to the commit history from
	// the external source code management service (e.g. GitHub).
	CommitService interface {
		// Find returns the commit information by sha.
		Find(ctx context.Context, user *User, repo, sha string) (*Commit, error)

		// FindRef returns the commit information by reference.
		FindRef(ctx context.Context, user *User, repo, ref string) (*Commit, error)

		// ListChanges returns the files change by sha or reference.
		ListChanges(ctx context.Context, user *User, repo, sha, ref string) ([]*Change, error)
	}
)
