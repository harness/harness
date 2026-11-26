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

package dockerhub

import (
	"encoding/json"
	"testing"

	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/require"
)

func TestRrmDockerhubGetImageName(t *testing.T) {
	a := &adapter{}
	name, err := a.GetImageName("nginx")
	require.NoError(t, err)
	require.Equal(t, "library/nginx", name)

	name, err = a.GetImageName("org/custom")
	require.NoError(t, err)
	require.Equal(t, "org/custom", name)
}

func TestRrmDockerhubNewClient(t *testing.T) {
	cli, err := NewClient(types.UpstreamProxy{})
	require.NoError(t, err)
	require.NotNil(t, cli)
}

func TestRrmDockerhubTypesJSON(t *testing.T) {
	resp := TokenResp{Token: "tok"}
	data, err := json.Marshal(resp)
	require.NoError(t, err)
	require.Contains(t, string(data), "tok")

	var cred LoginCredential
	require.NoError(t, json.Unmarshal([]byte(`{"username":"u","password":"p"}`), &cred))
	require.Equal(t, "u", cred.User)
	require.Equal(t, "p", cred.Password)
}
