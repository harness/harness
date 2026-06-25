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

package linkedpr

import "github.com/harness/gitness/types/enum"

// PullRequestPayload is the authoritative PR state decoded from the
// producer's ParseWebhookResponse. The handler reads it directly; no SCM
// API re-fetch. PR identity is (provider, repo provider id, Number) —
// the repo provider id is sourced from the matched LinkedRepo.
type PullRequestPayload struct {
	Number int

	Title       string
	Description string

	HeadRef string
	HeadSHA string
	BaseRef string
	BaseSHA string

	State enum.PullReqState
	Draft bool

	// CreatedAt / UpdatedAt are millis since epoch; UpdatedAt drives the
	// out-of-order guard and also stands in for closed_at / merged_at
	// (the parsed-response proto does not expose those).
	CreatedAt int64
	UpdatedAt int64

	HTMLURL string

	Author User
	// Sender is the upstream actor who triggered this event (the merger on
	// a merge action).
	Sender User

	Repository Repository
}

func (PullRequestPayload) Kind() Kind               { return KindPullRequest }
func (p PullRequestPayload) RepoProviderID() string { return p.Repository.ProviderID }
