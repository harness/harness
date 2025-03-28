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
	ObjectUser        Object = "user"
	ObjectRepository  Object = "repo"
	ObjectPullRequest Object = "pr"
	ObjectRule        Object = "rule"
)

type Verb string

// User verbs.
const (
	VerbUserCreate Verb = "create"
	VerbUserLogin  Verb = "login"
)

// Repository verbs.
const (
	VerbRepoCreate Verb = "create"
	VerbRepoPush   Verb = "push"
	VerbRepoDelete Verb = "delete"
)

// Pull request verbs.
const (
	VerbPullReqCreate  Verb = "create"
	VerbPullReqMerge   Verb = "merge"
	VerbPullReqClose   Verb = "close"
	VerbPullReqReopen  Verb = "reopen"
	VerbPullReqComment Verb = "comment"
)

// Rule verbs.
const (
	VerbRuleCreate Verb = "create"
)

type Submitter interface {
	// SubmitGroups should be called once a day to update info about all the groups.
	SubmitGroups(ctx context.Context) error

	// Submit submits an event.
	Submit(
		ctx context.Context,
		user *types.PrincipalInfo,
		object Object,
		verb Verb,
		properties map[string]any,
	) error
}
