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

package connector

import (
	"github.com/harness/gitness/app/connector/scm"
	"github.com/harness/gitness/app/store"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideConnectorHandler,
	ProvideSCMConnectorHandler,
)

// ProvideConnectorHandler provides a connector handler for handling connector-related ops.
func ProvideConnectorHandler(
	secretStore store.SecretStore,
	scmService *scm.Service,
) *Service {
	return New(secretStore, scmService)
}

// ProvideSCMConnectorHandler provides a SCM connector handler for specifically handling
// SCM connector related ops.
func ProvideSCMConnectorHandler(secretStore store.SecretStore) *scm.Service {
	return scm.NewService(secretStore)
}
