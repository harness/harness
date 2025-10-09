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

package scm

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type Service struct {
	secretStore store.SecretStore
}

func NewService(secretStore store.SecretStore) *Service {
	return &Service{
		secretStore: secretStore,
	}
}

func (s *Service) Test(ctx context.Context, c *types.Connector) (types.ConnectorTestResponse, error) {
	if !c.Type.IsSCM() {
		return types.ConnectorTestResponse{}, fmt.Errorf("connector type: %s is not an SCM connector", c.Type.String())
	}
	client, err := getSCMProvider(ctx, c, s.secretStore)
	if err != nil {
		return types.ConnectorTestResponse{}, err
	}
	// Check whether a valid user exists - if yes, the connection is successful
	_, _, err = client.Users.Find(ctx)
	if err != nil {
		return types.ConnectorTestResponse{Status: enum.ConnectorStatusFailed, ErrorMsg: err.Error()}, nil //nolint:nilerr
	}
	return types.ConnectorTestResponse{Status: enum.ConnectorStatusSuccess}, nil
}
