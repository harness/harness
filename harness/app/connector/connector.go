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
	"context"
	"time"

	"github.com/harness/gitness/app/connector/scm"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
)

var (
	testConnectionTimeout = 5 * time.Second
)

type Service struct {
	secretStore store.SecretStore
	// A separate SCM connector service is helpful here since the go-scm library abstracts out all the specific
	// SCM interactions, making the interfacing common for all the SCM connectors.
	// There might be connectors (eg docker, gcr, etc) in the future which require separate implementations.
	// Nevertheless, there should be an attempt to abstract out common functionality for different connector
	// types if possible - otherwise separate implementations can be written here.
	scmService *scm.Service
}

func New(secretStore store.SecretStore, scmService *scm.Service) *Service {
	return &Service{
		secretStore: secretStore,
		scmService:  scmService,
	}
}

func (s *Service) Test(
	ctx context.Context,
	connector *types.Connector,
) (types.ConnectorTestResponse, error) {
	// Set a timeout while testing connection.
	ctxWithTimeout, cancel := context.WithDeadline(ctx, time.Now().Add(testConnectionTimeout))
	defer cancel()
	if connector.Type.IsSCM() {
		return s.scmService.Test(ctxWithTimeout, connector)
	}
	return types.ConnectorTestResponse{}, nil
}
