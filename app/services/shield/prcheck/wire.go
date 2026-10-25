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

package prcheck

import (
	"context"

	checkevents "github.com/harness/gitness/app/events/check"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/shield"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideService,
)

// ProvideService provides the shield/security check service and, if Shield is enabled, registers
// its pull request listeners.
func ProvideService(
	ctx context.Context,
	config *types.Config,
	shieldSvc *shield.Service,
	pullreqStore store.PullReqStore,
	repoFinder refcache.RepoFinder,
	checkStore store.CheckStore,
	checkReporter *checkevents.Reporter,
	pullreqReaderFactory *events.ReaderFactory[*pullreqevents.Reader],
) (*Service, error) {
	svc := New(shieldSvc, pullreqStore, repoFinder, checkStore, checkReporter)
	if err := RegisterEventListeners(ctx, config.InstanceID, shieldSvc.Enabled(), pullreqReaderFactory, svc); err != nil {
		return nil, err
	}
	return svc, nil
}
