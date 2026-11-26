//  Copyright 2023 Harness, Inc.
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

package audit

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

type rpfSpaceCache struct {
	spaces map[int64]*types.SpaceCore
}

func (c *rpfSpaceCache) Stats() (int64, int64) { return 0, 0 }

func (c *rpfSpaceCache) Get(_ context.Context, key int64) (*types.SpaceCore, error) {
	s, ok := c.spaces[key]
	if !ok {
		return nil, errors.New("space not found")
	}
	return s, nil
}

func (c *rpfSpaceCache) Evict(context.Context, int64) {}

type rpfLoggedEvent struct {
	user      types.Principal
	resource  audit.Resource
	action    audit.Action
	spacePath string
	event     audit.Event
}

type rpfFakeAuditService struct {
	audit.Service
	events []rpfLoggedEvent
	err    error
}

func (s *rpfFakeAuditService) Log(
	_ context.Context, user types.Principal, resource audit.Resource, action audit.Action,
	spacePath string, options ...audit.Option,
) error {
	e := audit.Event{}
	for _, o := range options {
		o.Apply(&e)
	}
	s.events = append(s.events, rpfLoggedEvent{user, resource, action, spacePath, e})
	return s.err
}

func rpfFinder() refcache.SpaceFinder {
	return refcache.NewSpaceFinder(&rpfSpaceCache{spaces: map[int64]*types.SpaceCore{
		5: {ID: 5, Path: "acme/team"},
	}}, nil, nil, cache.Evictor[*types.SpaceCore]{})
}

func rpfInfo(kind *artifact.ArtifactType) pkg.ArtifactInfo {
	return pkg.ArtifactInfo{
		BaseInfo:      &pkg.BaseInfo{ParentID: 5},
		RegIdentifier: "my-reg",
		Image:         "demo",
		ArtifactType:  kind,
	}
}

func rpfAuthedContext() context.Context {
	return request.WithAuthSession(context.Background(), &auth.Session{
		Principal: types.Principal{ID: 3, UID: "alice"},
	})
}

func TestRpfLogArtifactUpload(t *testing.T) {
	kind := artifact.ArtifactType("model")

	t.Run("logs a versioned upload", func(t *testing.T) {
		svc := &rpfFakeAuditService{}
		LogArtifactUpload(rpfAuthedContext(), svc, rpfFinder(), rpfInfo(&kind), "1.0", "img-uuid", "art-uuid")
		require.Len(t, svc.events, 1)
		ev := svc.events[0]
		require.Equal(t, audit.ActionUploaded, ev.action)
		require.Equal(t, "acme/team", ev.spacePath)
		require.Equal(t, "alice", ev.user.UID)
		require.Equal(t, audit.ResourceTypeRegistryArtifact, ev.resource.Type)
		require.Equal(t, "demo:1.0", ev.resource.Identifier)
		require.Equal(t, "art-uuid", ev.resource.Data[AuditKeyArtifactUUID])
		require.Equal(t, "my-reg", ev.resource.Data[AuditKeyRegistryName])
		require.Equal(t, "demo", ev.resource.Data[AuditKeyPackageName])
		require.Equal(t, "model", ev.resource.Data[AuditKeyPackageKind])
		require.Equal(t, "1.0", ev.resource.Data[AuditKeyVersionName])
		require.Equal(t, "img-uuid", ev.event.Data[AuditKeyImageUUID])
	})

	t.Run("version-less upload uses the bare package name", func(t *testing.T) {
		svc := &rpfFakeAuditService{}
		LogArtifactUpload(rpfAuthedContext(), svc, rpfFinder(), rpfInfo(nil), "", "i", "a")
		require.Len(t, svc.events, 1)
		require.Equal(t, "demo", svc.events[0].resource.Identifier)
		require.Empty(t, svc.events[0].resource.Data[AuditKeyPackageKind])
	})

	t.Run("no auth session", func(t *testing.T) {
		svc := &rpfFakeAuditService{}
		LogArtifactUpload(context.Background(), svc, rpfFinder(), rpfInfo(nil), "1.0", "i", "a")
		require.Empty(t, svc.events)
	})

	t.Run("unknown parent space", func(t *testing.T) {
		svc := &rpfFakeAuditService{}
		info := rpfInfo(nil)
		info.ParentID = 404
		LogArtifactUpload(rpfAuthedContext(), svc, rpfFinder(), info, "1.0", "i", "a")
		require.Empty(t, svc.events)
	})

	t.Run("audit failure is swallowed", func(t *testing.T) {
		svc := &rpfFakeAuditService{err: errors.New("audit down")}
		require.NotPanics(t, func() {
			LogArtifactUpload(rpfAuthedContext(), svc, rpfFinder(), rpfInfo(nil), "1.0", "i", "a")
		})
		require.Len(t, svc.events, 1)
	})
}

func TestRpfLogArtifactDownload(t *testing.T) {
	kind := artifact.ArtifactType("dataset")

	t.Run("logs a versioned download", func(t *testing.T) {
		svc := &rpfFakeAuditService{}
		LogArtifactDownload(rpfAuthedContext(), svc, rpfFinder(), rpfInfo(&kind), "2.0")
		require.Len(t, svc.events, 1)
		ev := svc.events[0]
		require.Equal(t, audit.ActionDownloaded, ev.action)
		require.Equal(t, "acme/team", ev.spacePath)
		require.Equal(t, "demo:2.0", ev.resource.Identifier)
		require.Equal(t, "dataset", ev.resource.Data[AuditKeyPackageKind])
		require.Empty(t, ev.resource.Data[AuditKeyArtifactUUID])
	})

	t.Run("version-less download", func(t *testing.T) {
		svc := &rpfFakeAuditService{}
		LogArtifactDownload(rpfAuthedContext(), svc, rpfFinder(), rpfInfo(nil), "")
		require.Equal(t, "demo", svc.events[0].resource.Identifier)
	})

	t.Run("no auth session", func(t *testing.T) {
		svc := &rpfFakeAuditService{}
		LogArtifactDownload(context.Background(), svc, rpfFinder(), rpfInfo(nil), "1")
		require.Empty(t, svc.events)
	})

	t.Run("unknown parent space", func(t *testing.T) {
		svc := &rpfFakeAuditService{}
		info := rpfInfo(nil)
		info.ParentID = 404
		LogArtifactDownload(rpfAuthedContext(), svc, rpfFinder(), info, "1")
		require.Empty(t, svc.events)
	})

	t.Run("audit failure is swallowed", func(t *testing.T) {
		svc := &rpfFakeAuditService{err: errors.New("audit down")}
		require.NotPanics(t, func() {
			LogArtifactDownload(rpfAuthedContext(), svc, rpfFinder(), rpfInfo(nil), "1")
		})
	})
}
