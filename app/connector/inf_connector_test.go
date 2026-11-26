// Copyright 2026 Harness, Inc.
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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harness/gitness/app/connector/scm"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

type infSecretStore struct {
	store.SecretStore
}

func (infSecretStore) FindByIdentifier(_ context.Context, _ int64, id string) (*types.Secret, error) {
	return &types.Secret{Identifier: id, Data: "token-value-0000"}, nil
}

func TestInfService_Test(t *testing.T) {
	ctx := context.Background()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": 1, "login": "alice"}`))
	}))
	defer srv.Close()

	svc := New(infSecretStore{}, scm.NewService(infSecretStore{}, false))

	t.Run("scm connectors are tested through the scm service", func(t *testing.T) {
		resp, err := svc.Test(ctx, &types.Connector{
			Type: enum.ConnectorTypeGithub,
			ConnectorConfig: types.ConnectorConfig{Github: &types.GithubConnectorData{
				APIURL: srv.URL + "/",
				Auth: &types.ConnectorAuth{
					AuthType: enum.ConnectorAuthTypeBearer,
					Bearer:   &types.BearerTokenCreds{Token: types.SecretRef{Identifier: "tok"}},
				},
			}},
		})
		require.NoError(t, err)
		require.Equal(t, enum.ConnectorStatusSuccess, resp.Status)
	})
	t.Run("scm failures are returned", func(t *testing.T) {
		_, err := svc.Test(ctx, &types.Connector{Type: enum.ConnectorTypeGithub})
		require.Error(t, err)
	})
	t.Run("other connector types are a no-op", func(t *testing.T) {
		resp, err := svc.Test(ctx, &types.Connector{Type: "docker"})
		require.NoError(t, err)
		require.Equal(t, types.ConnectorTestResponse{}, resp)
	})
}
