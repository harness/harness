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

package space

import (
	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/exporter"
	"github.com/harness/gitness/app/services/gitspace"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/app/services/label"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideController,
)

func ProvideController(config *types.Config, tx dbtx.Transactor, urlProvider url.Provider, sseStreamer sse.Streamer,
	identifierCheck check.SpaceIdentifier, authorizer authz.Authorizer, spacePathStore store.SpacePathStore,
	pipelineStore store.PipelineStore, secretStore store.SecretStore,
	connectorStore store.ConnectorStore, templateStore store.TemplateStore,
	spaceStore store.SpaceStore, repoStore store.RepoStore, principalStore store.PrincipalStore,
	repoCtrl *repo.Controller, membershipStore store.MembershipStore, importer *importer.Repository,
	exporter *exporter.Repository, limiter limiter.ResourceLimiter, publicAccess publicaccess.Service,
	auditService audit.Service, gitspaceService *gitspace.Service,
	labelSvc *label.Service,
) *Controller {
	return NewController(config, tx, urlProvider, sseStreamer, identifierCheck, authorizer,
		spacePathStore, pipelineStore, secretStore,
		connectorStore, templateStore,
		spaceStore, repoStore, principalStore,
		repoCtrl, membershipStore, importer,
		exporter, limiter, publicAccess,
		auditService, gitspaceService,
		labelSvc)
}
