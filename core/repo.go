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

// Repository visibility.
const (
	VisibilityPublic   = "public"
	VisibilityPrivate  = "private"
	VisibilityInternal = "internal"
)

// Version control systems.
const (
	VersionControlGit       = "git"
	VersionControlMercurial = "hg"
)

type (
	// Repository represents a source code repository.
	Repository struct {
		ID            int64  `json:"id"`
		UID           string `json:"uid"`
		UserID        int64  `json:"user_id"`
		Namespace     string `json:"namespace"`
		Name          string `json:"name"`
		Slug          string `json:"slug"`
		SCM           string `json:"scm"`
		HTTPURL       string `json:"git_http_url"`
		SSHURL        string `json:"git_ssh_url"`
		Link          string `json:"link"`
		Branch        string `json:"default_branch"`
		Private       bool   `json:"private"`
		Visibility    string `json:"visibility"`
		Active        bool   `json:"active"`
		Config        string `json:"config_path"`
		Trusted       bool   `json:"trusted"`
		Protected     bool   `json:"protected"`
		IgnoreForks   bool   `json:"ignore_forks"`
		IgnorePulls   bool   `json:"ignore_pull_requests"`
		CancelPulls   bool   `json:"auto_cancel_pull_requests"`
		CancelPush    bool   `json:"auto_cancel_pushes"`
		CancelRunning bool   `json:"auto_cancel_running"`
		Timeout       int64  `json:"timeout"`
		Throttle      int64  `json:"throttle,omitempty"`
		Counter       int64  `json:"counter"`
		Synced        int64  `json:"synced"`
		Created       int64  `json:"created"`
		Updated       int64  `json:"updated"`
		Version       int64  `json:"version"`
		Signer        string `json:"-"`
		Secret        string `json:"-"`
		Build         *Build `json:"build,omitempty"`
		Perms         *Perm  `json:"permissions,omitempty"`
		Archived      bool   `json:"archived"`
	}

	RepoBuildStage struct {
		RepoNamespace     string `json:"repo_namespace"`
		RepoName          string `json:"repo_name"`
		RepoSlug          string `json:"repo_slug"`
		BuildNumber       int64  `json:"build_number"`
		BuildAuthor       string `json:"build_author"`
		BuildAuthorName   string `json:"build_author_name"`
		BuildAuthorEmail  string `json:"build_author_email"`
		BuildAuthorAvatar string `json:"build_author_avatar"`
		BuildSender       string `json:"build_sender"`
		BuildStarted      int64  `json:"build_started"`
		BuildFinished     int64  `json:"build_finished"`
		BuildCreated      int64  `json:"build_created"`
		BuildUpdated      int64  `json:"build_updated"`
		StageName         string `json:"stage_name"`
		StageKind         string `json:"stage_kind"`
		StageType         string `json:"stage_type"`
		StageStatus       string `json:"stage_status"`
		StageMachine      string `json:"stage_machine"`
		StageOS           string `json:"stage_os"`
		StageArch         string `json:"stage_arch"`
		StageVariant      string `json:"stage_variant"`
		StageKernel       string `json:"stage_kernel"`
		StageLimit        string `json:"stage_limit"`
		StageLimitRepo    string `json:"stage_limit_repo"`
		StageStarted      int64  `json:"stage_started"`
		StageStopped      int64  `json:"stage_stopped"`
	}

	// RepositoryStore defines operations for working with repositories.
	RepositoryStore interface {
		// List returns a repository list from the datastore.
		List(context.Context, int64) ([]*Repository, error)

		// ListLatest returns a unique repository list form
		// the datastore with the most recent build.
		ListLatest(context.Context, int64) ([]*Repository, error)

		// ListRecent returns a non-unique repository list form
		// the datastore with the most recent builds.
		ListRecent(context.Context, int64) ([]*Repository, error)

		// ListIncomplete returns a non-unique repository list form
		// the datastore with incomplete builds.
		ListIncomplete(context.Context) ([]*Repository, error)

		// ListRunningStatus returns a list of build / repository /stage information for builds that are incomplete.
		ListRunningStatus(context.Context) ([]*RepoBuildStage, error)

		// ListAll returns a paginated list of all repositories
		// stored in the database, including disabled repositories.
		ListAll(ctx context.Context, limit, offset int) ([]*Repository, error)

		// Find returns a repository from the datastore.
		Find(context.Context, int64) (*Repository, error)

		// FindName returns a named repository from the datastore.
		FindName(context.Context, string, string) (*Repository, error)

		// Create persists a new repository in the datastore.
		Create(context.Context, *Repository) error

		// Activate persists the activated repository to the datastore.
		Activate(context.Context, *Repository) error

		// Update persists repository changes to the datastore.
		Update(context.Context, *Repository) error

		// Delete deletes a repository from the datastore.
		Delete(context.Context, *Repository) error

		// Count returns a count of activated repositories.
		Count(context.Context) (int64, error)

		// Increment returns an incremented build number
		Increment(context.Context, *Repository) (*Repository, error)
	}

	// RepositoryService provides access to repository information
	// in the remote source code management system (e.g. GitHub).
	RepositoryService interface {
		// List returns a list of repositories.
		List(ctx context.Context, user *User) ([]*Repository, error)

		// Find returns the named repository details.
		Find(ctx context.Context, user *User, repo string) (*Repository, error)

		// FindPerm returns the named repository permissions.
		FindPerm(ctx context.Context, user *User, repo string) (*Perm, error)
	}
)
