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

package branch

import (
	"context"

	gitevents "github.com/harness/gitness/app/events/git"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideService,
)

// ProvideService creates a new branch service.
func ProvideService(
	ctx context.Context,
	config Config,
	branchStore store.BranchStore,
	pullReqStore store.PullReqStore,
	gitReaderFactory *events.ReaderFactory[*gitevents.Reader],
	pullreqReaderFactory *events.ReaderFactory[*pullreqevents.Reader],
) (*Service, error) {
	return New(
		ctx,
		config,
		branchStore,
		gitReaderFactory,
		pullreqReaderFactory,
		pullReqStore,
	)
}
