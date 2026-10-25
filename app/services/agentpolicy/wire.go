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

package agentpolicy

import (
	"github.com/harness/gitness/app/services/agentguard"
	"github.com/harness/gitness/app/services/aidecide"
	"github.com/harness/gitness/app/services/aigateway"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideEvaluator,
)

// ProvideEvaluator provides the fix agent launch policy.
func ProvideEvaluator(
	config *types.Config,
	modes *aigateway.ModeResolver,
	guard *agentguard.Guard,
	gateway *aigateway.Gateway,
	decide *aidecide.Service,
) *Evaluator {
	return New(NewConfig(config), modes, guard, gateway.Planner(), decide)
}
