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

package auth

import (
	"testing"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestHndCheckResourceWrappers(t *testing.T) {
	ctx := t.Context()
	authz := hndAuthorizer{mode: hndAuthAllow}
	session := hndSession(false)

	require.NoError(t, CheckSecret(ctx, authz, session, "root", "s1", enum.PermissionSecretView))
	require.NoError(t, CheckPipeline(ctx, authz, session, "root/repo1", "pipe1", enum.PermissionPipelineView))
	require.NoError(t, CheckConnector(ctx, authz, session, "root", "conn1", enum.PermissionConnectorView))
	require.NoError(t, CheckTemplate(ctx, authz, session, "root", "tmpl1", enum.PermissionTemplateView))
	require.NoError(t, CheckService(ctx, authz, session, &types.Service{UID: "svc1"}, enum.PermissionServiceView))
	space := &types.Space{ID: 1, Path: "root", Identifier: "root"}
	require.NoError(t, CheckServiceAccount(ctx, authz, session,
		hndSpaceStore{space: space}, hndRepoStore{}, enum.ParentResourceTypeSpace, 1,
		"sa1", enum.PermissionServiceAccountView))
	require.NoError(t, CheckRegistry(ctx, authz, session, types.PermissionCheck{
		Scope: types.Scope{SpacePath: "root"}, Resource: types.Resource{
			Type: enum.ResourceTypeRegistry, Identifier: "reg1",
		}, Permission: enum.PermissionRegistryView,
	}))
	require.NoError(t, CheckInfraProvider(ctx, authz, session, "root", "ip1", enum.PermissionInfraProviderView))
	require.NoError(t, CheckGitspace(ctx, authz, session, "root", "gs1", enum.PermissionGitspaceView))
}

func TestHndCheckPipelineBadPath(t *testing.T) {
	ctx := t.Context()
	err := CheckPipeline(ctx, hndAuthorizer{mode: hndAuthAllow}, hndSession(false), "", "p", enum.PermissionPipelineView)
	require.Error(t, err)
}
