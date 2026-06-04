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

package types

// LinkedPullReq holds provider-side metadata for a pull request mirrored
// from an external SCM. State / branches / SHAs / PR number live on the
// parent pullreqs row (joined via PullReqID); the upstream PR number is
// mirrored into pullreq_number on create.
type LinkedPullReq struct {
	PullReqID int64

	ProviderType string
	// ProviderRepoID is the upstream repo's SCM id (mirror of
	// LinkedRepo.ProviderRepoID).
	ProviderRepoID string
	ProviderURL    string

	ProviderAuthorLogin     string
	ProviderAuthorAvatarURL string
	ProviderAuthorURL       string

	// ProviderUpdatedAt is the provider-clock "updated_at"; drives the
	// out-of-order webhook guard.
	ProviderUpdatedAt int64

	// MergerLogin is the provider-side login of the user who merged this PR.
	MergerLogin string

	LastSyncedAt int64
}
