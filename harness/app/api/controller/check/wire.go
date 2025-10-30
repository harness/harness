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

package check

import (
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	checkevents "github.com/harness/gitness/app/events/check"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types/enum"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideCheckSanitizers,
	ProvideController,
)

func ProvideController(
	tx dbtx.Transactor,
	authorizer authz.Authorizer,
	spaceStore store.SpaceStore,
	checkStore store.CheckStore,
	spaceFinder refcache.SpaceFinder,
	repoFinder refcache.RepoFinder,
	git git.Interface,
	sanitizers map[enum.CheckPayloadKind]func(in *ReportInput, s *auth.Session) error,
	sseStreamer sse.Streamer,
	eventReporter *checkevents.Reporter,
) *Controller {
	return NewController(
		tx,
		authorizer,
		spaceStore,
		checkStore,
		spaceFinder,
		repoFinder,
		git,
		sanitizers,
		sseStreamer,
		eventReporter,
	)
}
