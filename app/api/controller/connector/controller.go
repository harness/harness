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
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/connector"
	"github.com/harness/gitness/app/store"
)

type Controller struct {
	connectorStore   store.ConnectorStore
	connectorService *connector.Service

	authorizer authz.Authorizer
	spaceStore store.SpaceStore
}

func NewController(
	authorizer authz.Authorizer,
	connectorStore store.ConnectorStore,
	connectorService *connector.Service,
	spaceStore store.SpaceStore,
) *Controller {
	return &Controller{
		connectorStore:   connectorStore,
		connectorService: connectorService,
		authorizer:       authorizer,
		spaceStore:       spaceStore,
	}
}
