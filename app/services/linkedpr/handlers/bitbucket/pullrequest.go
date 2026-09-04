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

package bitbucket

import (
	"fmt"

	"github.com/harness/gitness/app/services/linkedpr"
	"github.com/harness/gitness/app/services/linkedpr/handlers"
)

// PullRequestHandler gives Wire a provider-specific type while delegating all
// provider-neutral mirroring behavior to handlers.PullRequestHandler.
type PullRequestHandler struct {
	*handlers.PullRequestHandler
}

// PRSyncSpec maps the source branch directly into the Harness Code PR ref.
// Bitbucket Cloud does not expose a server-side refs/pull namespace, so the
// PR head is whatever the mutable source branch points at when we fetch it.
func PRSyncSpec(p linkedpr.PullRequestPayload) []handlers.RefSyncEntry {
	return []handlers.RefSyncEntry{
		{RemoteRef: fmt.Sprintf("refs/heads/%s", p.BaseRef)},
		{
			RemoteRef: fmt.Sprintf("refs/heads/%s", p.HeadRef),
			LocalRef:  fmt.Sprintf("refs/pullreq/%d/head", p.Number),
		},
	}
}
