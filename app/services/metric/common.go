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

package metric

import (
	"context"

	"github.com/harness/gitness/types"
)

type Object string

const (
	ObjectRepository  Object = "repository"
	ObjectPullRequest Object = "pull_request"
)

type VerbRepo string

// Repository verbs.
const (
	VerbRepoCreate VerbRepo = "create"
	VerbRepoUpdate VerbRepo = "update"
	VerbRepoDelete VerbRepo = "delete"
)

type VerbPullReq string

// Pull request verbs.
const (
	VerbPullReqCreate VerbPullReq = "create"
	VerbPullReqMerge  VerbPullReq = "merge"
	VerbPullReqClose  VerbPullReq = "close"
	VerbPullReqReopen VerbPullReq = "reopen"
)

type Submitter interface {
	// SubmitGroups should be called once a day to update info about all the groups.
	SubmitGroups(ctx context.Context) error

	// SubmitForRepo submits an event for a repository.
	SubmitForRepo(
		ctx context.Context,
		user *types.PrincipalInfo,
		verb VerbRepo,
		properties map[string]any,
	) error

	// SubmitForPullReq submits an event for a pull request.
	SubmitForPullReq(
		ctx context.Context,
		user *types.PrincipalInfo,
		verb VerbPullReq,
		properties map[string]any,
	) error
}
