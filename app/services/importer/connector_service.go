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

package importer

import (
	"context"

	"github.com/harness/gitness/errors"
)

// ConnectorService bundles operations against the platform's connector store:
// resolving auth/clone info for a given connector, looking up the provider's
// stable repo identifier and type, plus translating between a connector
// reference (whatever scope-prefixed form the caller uses) and the (connector
// space path, identifier) pair that linked_repositories stores verbatim. The
// default noop treats refs as identifiers under the parent space; downstream
// platforms install a real implementation that understands their scoping rules.
//
// FetchProviderRepoInfo is split from GetAccessInfo because it's a
// link-creation-time concern: only LinkedCreate needs the provider's stable
// repo id, while GetAccessInfo is on the hot path (sync jobs, credential
// refresh) where an extra SCM round-trip would be wasted work and an extra
// failure surface.
type ConnectorService interface {
	GetAccessInfo(ctx context.Context, c ConnectorDef) (AccessInfo, error)
	FetchProviderRepoInfo(ctx context.Context, c ConnectorDef) (ProviderRepoInfo, error)
	ResolveConnectorRef(parentSpacePath, ref string) (connectorPath, connectorIdentifier string)
	EncodeConnectorRef(parentSpacePath, connectorPath, connectorIdentifier string) string
}

// ProviderRepoInfo carries the stable identifier/type pair the provider
// returns for a linked repository, persisted on the linked_repositories row
// so webhook ingress can disambiguate ids that collide across providers.
type ProviderRepoInfo struct {
	RepoID string
	Type   ProviderType
}

type connectorServiceNoop struct{}

func (connectorServiceNoop) GetAccessInfo(context.Context, ConnectorDef) (AccessInfo, error) {
	return AccessInfo{}, errors.InvalidArgument("This feature is not supported.")
}

func (connectorServiceNoop) FetchProviderRepoInfo(context.Context, ConnectorDef) (ProviderRepoInfo, error) {
	return ProviderRepoInfo{}, errors.InvalidArgument("This feature is not supported.")
}

func (connectorServiceNoop) ResolveConnectorRef(parentSpacePath, ref string) (string, string) {
	return parentSpacePath, ref
}

func (connectorServiceNoop) EncodeConnectorRef(_, _, connectorIdentifier string) string {
	return connectorIdentifier
}
