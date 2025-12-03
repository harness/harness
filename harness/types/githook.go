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

import (
	"github.com/harness/gitness/git/hook"
)

// GithookInputBase contains the base input of the githook apis.
type GithookInputBase struct {
	RepoID      int64
	PrincipalID int64
	Internal    bool // Internal calls originate from Gitness, and external calls are direct git pushes.
}

// GithookPreReceiveInput is the input for the pre-receive githook api call.
type GithookPreReceiveInput struct {
	GithookInputBase
	hook.PreReceiveInput
}

// GithookUpdateInput is the input for the update githook api call.
type GithookUpdateInput struct {
	GithookInputBase
	hook.UpdateInput
}

// GithookPostReceiveInput is the input for the post-receive githook api call.
type GithookPostReceiveInput struct {
	GithookInputBase
	hook.PostReceiveInput
}
