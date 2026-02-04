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

package udp

import (
	"context"

	"github.com/harness/gitness/types"
)

// Action constants.
const (
	ActionRegistryCreated = "REGISTRY_CREATED"
	ActionRegistryUpdated = "REGISTRY_UPDATED"
	ActionRegistryDeleted = "REGISTRY_DELETED"
)

// Resource type constants.
const (
	ResourceTypeRegistryUpstreamProxy = "registry_upstream_proxy"
	ResourceTypeRegistryVirtual       = "registry_virtual"
)

type Service interface {
	InsertEvent(
		ctx context.Context,
		action string,
		resourceType string,
		resourceIdentifier string,
		parentRef string,
		principal types.Principal,
		newObject interface{},
		oldObject interface{},
	)
}
