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

package base

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/quarantine"
	"github.com/harness/gitness/registry/app/pkg/response"
	commonstype "github.com/harness/gitness/registry/app/pkg/types/commons"
	huggingfacetypes "github.com/harness/gitness/registry/app/pkg/types/huggingface"
	"github.com/harness/gitness/registry/app/store"
	registrytypes "github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/require"
)

type rpfStubArtifact struct {
	packageTypes []artifact.PackageType
	registryType artifact.RegistryType
}

func (a rpfStubArtifact) GetArtifactType() artifact.RegistryType { return a.registryType }

func (a rpfStubArtifact) GetPackageTypes() []artifact.PackageType { return a.packageTypes }

type rpfRegistryRepo struct {
	store.RegistryRepository
	byName    map[string]registrytypes.Registry
	upstreams []registrytypes.Registry
	idsErr    error
	gotIDs    []int64
}

func (r *rpfRegistryRepo) GetByParentIDAndName(
	_ context.Context, _ int64, name string, _ ...registrytypes.QueryOption,
) (*registrytypes.Registry, error) {
	reg, ok := r.byName[name]
	if !ok {
		return nil, errors.New("not found")
	}
	return &reg, nil
}

func (r *rpfRegistryRepo) GetByIDIn(
	_ context.Context, ids []int64, _ ...registrytypes.QueryOption,
) (*[]registrytypes.Registry, error) {
	r.gotIDs = ids
	if r.idsErr != nil {
		return nil, r.idsErr
	}
	return &r.upstreams, nil
}

type rpfQuarantineFinder struct {
	quarantine.Finder
	err      error
	gotTypes []*artifact.ArtifactType
}

func (f *rpfQuarantineFinder) CheckArtifactQuarantineStatus(
	_ context.Context, _ int64, _ string, _ string, artifactType *artifact.ArtifactType,
) error {
	f.gotTypes = append(f.gotTypes, artifactType)
	return f.err
}

type rpfFirewall struct {
	err   error
	calls int
}

func (f *rpfFirewall) CheckPolicyViolation(
	context.Context, int64, string, string, *artifact.ArtifactType,
) error {
	f.calls++
	return f.err
}

type rpfResponse struct {
	err  error
	from string
	hits []any
}

func (r *rpfResponse) GetError() error { return r.err }

var _ response.Response = (*rpfResponse)(nil)

func rpfInfo() pkg.PackageArtifactInfo {
	return commonstype.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{
			BaseInfo:      &pkg.BaseInfo{ParentID: 1},
			RegIdentifier: "virtual",
			Image:         "demo",
		},
		Version: "1.0",
	}
}

// rpfRegisterStubs registers local and remote artifact implementations for the generic package type and
// restores the global registry afterwards.
func rpfRegisterStubs(t *testing.T) (local rpfStubArtifact, remote rpfStubArtifact) {
	t.Helper()
	saved := TypeRegistry
	TypeRegistry = map[string]pkg.Artifact{}
	t.Cleanup(func() { TypeRegistry = saved })
	local = rpfStubArtifact{
		packageTypes: []artifact.PackageType{artifact.PackageTypeGENERIC},
		registryType: artifact.RegistryTypeVIRTUAL,
	}
	remote = rpfStubArtifact{
		packageTypes: []artifact.PackageType{artifact.PackageTypeGENERIC},
		registryType: artifact.RegistryTypeUPSTREAM,
	}
	Register(local, remote)
	return local, remote
}

func rpfRegistries() *rpfRegistryRepo {
	return &rpfRegistryRepo{
		byName: map[string]registrytypes.Registry{
			"virtual": {
				ID: 1, Name: "virtual", Type: artifact.RegistryTypeVIRTUAL, PackageType: artifact.PackageTypeGENERIC,
				UpstreamProxies: []int64{20, 99, 10},
			},
			"plain": {ID: 2, Name: "plain", Type: artifact.RegistryTypeVIRTUAL, PackageType: artifact.PackageTypeGENERIC},
			"blocked": {
				ID: 3, Name: "blocked", Type: artifact.RegistryTypeVIRTUAL, PackageType: artifact.PackageTypeGENERIC,
				BlockedPattern: []string{"^demo:"},
			},
			"allowlist": {
				ID: 4, Name: "allowlist", Type: artifact.RegistryTypeVIRTUAL, PackageType: artifact.PackageTypeGENERIC,
				AllowedPattern: []string{"^other:"},
			},
		},
		upstreams: []registrytypes.Registry{
			{ID: 10, Name: "up-b", Type: artifact.RegistryTypeUPSTREAM, PackageType: artifact.PackageTypeGENERIC},
			{ID: 20, Name: "up-a", Type: artifact.RegistryTypeUPSTREAM, PackageType: artifact.PackageTypeGENERIC},
		},
	}
}

func rpfInfoFor(registry string) pkg.PackageArtifactInfo {
	info, ok := rpfInfo().(commonstype.ArtifactInfo)
	if !ok {
		return nil
	}
	info.RegIdentifier = registry
	return info
}

func TestRpfRegisterAndLookup(t *testing.T) {
	local, remote := rpfRegisterStubs(t)
	require.Equal(t, local, GetRegistry(artifact.PackageTypeGENERIC, artifact.RegistryTypeVIRTUAL))
	require.Equal(t, remote, GetRegistry(artifact.PackageTypeGENERIC, artifact.RegistryTypeUPSTREAM))
	require.Nil(t, GetRegistry(artifact.PackageTypeNPM, artifact.RegistryTypeVIRTUAL))
	require.Equal(t, remote, GetArtifactRegistry(registrytypes.Registry{
		PackageType: artifact.PackageTypeGENERIC, Type: artifact.RegistryTypeUPSTREAM,
	}))
	require.Equal(t, "GENERIC:VIRTUAL", getFactoryKey(artifact.PackageTypeGENERIC, artifact.RegistryTypeVIRTUAL))
}

func TestRpfGetOrderedRepos(t *testing.T) {
	ctx := context.Background()

	got, err := GetOrderedRepos(ctx, rpfRegistries(), "virtual", 1, false)
	require.NoError(t, err)
	require.Len(t, got, 1)

	repo := rpfRegistries()
	got, err = GetOrderedRepos(ctx, repo, "virtual", 1, true)
	require.NoError(t, err)
	require.Equal(t, []int64{20, 99, 10}, repo.gotIDs)
	names := make([]string, 0, len(got))
	for _, r := range got {
		names = append(names, r.Name)
	}
	require.Equal(t, []string{"virtual", "up-a", "up-b"}, names)

	got, err = GetOrderedRepos(ctx, rpfRegistries(), "plain", 1, true)
	require.NoError(t, err)
	require.Len(t, got, 1)

	_, err = GetOrderedRepos(ctx, rpfRegistries(), "missing", 1, true)
	require.ErrorContains(t, err, "registry missing not found")

	failing := rpfRegistries()
	failing.idsErr = errors.New("db down")
	got, err = GetOrderedRepos(ctx, failing, "virtual", 1, true)
	require.ErrorContains(t, err, "db down")
	require.Len(t, got, 1)
}

func TestRpfProxyWrapper(t *testing.T) {
	ctx := context.Background()
	rpfRegisterStubs(t)

	var visited []string
	succeedOn := func(target string) func(registrytypes.Registry, pkg.Artifact) response.Response {
		return func(r registrytypes.Registry, _ pkg.Artifact) response.Response {
			visited = append(visited, r.Name)
			if r.Name == target {
				return &rpfResponse{from: r.Name}
			}
			return &rpfResponse{err: errors.New("miss in " + r.Name)}
		}
	}

	t.Run("falls through to the first registry that succeeds", func(t *testing.T) {
		visited = nil
		got, err := ProxyWrapper(ctx, rpfRegistries(), nil, succeedOn("up-b"), rpfInfo(), false)
		require.NoError(t, err)
		rr, ok := got.(*rpfResponse)
		require.True(t, ok)
		require.Equal(t, "up-b", rr.from)
		require.Equal(t, []string{"virtual", "up-a", "up-b"}, visited)
	})

	t.Run("no proxy wrapper stays on the requested registry", func(t *testing.T) {
		visited = nil
		_, err := NoProxyWrapper(ctx, rpfRegistries(), succeedOn("up-b"), rpfInfo())
		require.ErrorContains(t, err, "miss in virtual")
		require.Equal(t, []string{"virtual"}, visited)
	})

	t.Run("all registries failing returns the last error", func(t *testing.T) {
		_, err := ProxyWrapper(ctx, rpfRegistries(), nil, succeedOn("none"), rpfInfo(), false)
		require.ErrorContains(t, err, "miss in up-b")
	})

	t.Run("unknown registry", func(t *testing.T) {
		_, err := ProxyWrapper(ctx, rpfRegistries(), nil, succeedOn("x"), rpfInfoFor("missing"), false)
		require.ErrorContains(t, err, "not found")
	})

	t.Run("registries without an implementation are skipped", func(t *testing.T) {
		saved := TypeRegistry
		TypeRegistry = map[string]pkg.Artifact{}
		defer func() { TypeRegistry = saved }()
		_, err := ProxyWrapper(ctx, rpfRegistries(), nil, succeedOn("virtual"), rpfInfo(), false)
		require.ErrorContains(t, err, "no matching artifacts found")
	})

	t.Run("allow and block patterns skip registries", func(t *testing.T) {
		_, err := ProxyWrapper(ctx, rpfRegistries(), nil, succeedOn("blocked"), rpfInfoFor("blocked"), false)
		require.ErrorContains(t, err, "skipped registries: [blocked]")
		_, err = ProxyWrapper(ctx, rpfRegistries(), nil, succeedOn("allowlist"), rpfInfoFor("allowlist"), false)
		require.ErrorContains(t, err, "skipped registries: [allowlist]")
	})

	t.Run("an info without image and version is not filtered", func(t *testing.T) {
		info := commonstype.ArtifactInfo{ArtifactInfo: pkg.ArtifactInfo{
			BaseInfo: &pkg.BaseInfo{ParentID: 1}, RegIdentifier: "blocked",
		}}
		got, err := ProxyWrapper(ctx, rpfRegistries(), nil, succeedOn("blocked"), info, false)
		require.NoError(t, err)
		require.NotNil(t, got)
	})
}

func TestRpfProxyWrapperChecks(t *testing.T) {
	ctx := context.Background()
	rpfRegisterStubs(t)
	ok := func(r registrytypes.Registry, _ pkg.Artifact) response.Response { return &rpfResponse{from: r.Name} }

	t.Run("quarantined artifacts stop the request", func(t *testing.T) {
		finder := &rpfQuarantineFinder{err: usererror.ErrQuarantinedArtifact}
		_, err := ProxyWrapper(ctx, rpfRegistries(), finder, ok, rpfInfo(), true)
		require.ErrorIs(t, err, usererror.ErrQuarantinedArtifact)

		finder = &rpfQuarantineFinder{err: errors.New("cache down")}
		_, err = ProxyWrapper(ctx, rpfRegistries(), finder, ok, rpfInfo(), true)
		require.ErrorContains(t, err, "cache down")
	})

	t.Run("quarantine is only consulted when enabled", func(t *testing.T) {
		finder := &rpfQuarantineFinder{err: usererror.ErrQuarantinedArtifact}
		_, err := ProxyWrapper(ctx, rpfRegistries(), finder, ok, rpfInfo(), false)
		require.NoError(t, err)
		require.Empty(t, finder.gotTypes)
	})

	t.Run("huggingface repo types are forwarded", func(t *testing.T) {
		finder := &rpfQuarantineFinder{}
		hf := huggingfacetypes.ArtifactInfo{
			ArtifactInfo: pkg.ArtifactInfo{BaseInfo: &pkg.BaseInfo{ParentID: 1}, RegIdentifier: "plain", Image: "m"},
			Repo:         "org/m", Revision: "main", RepoType: artifact.ArtifactType("model"),
		}
		_, err := ProxyWrapper(ctx, rpfRegistries(), finder, ok, hf, true)
		require.NoError(t, err)
		require.Len(t, finder.gotTypes, 1)
		require.NotNil(t, finder.gotTypes[0])
		require.Equal(t, artifact.ArtifactType("model"), *finder.gotTypes[0])
	})

	t.Run("dependency firewall only applies to upstream registries", func(t *testing.T) {
		firewall := &rpfFirewall{err: usererror.ErrArtifactBlocked}
		// The virtual registry succeeds first, so the firewall is never reached.
		_, err := ProxyWrapperWithChecks(ctx, rpfRegistries(), nil, firewall, ok, rpfInfo(), false, true)
		require.NoError(t, err)
		require.Zero(t, firewall.calls)

		fail := func(r registrytypes.Registry, _ pkg.Artifact) response.Response {
			return &rpfResponse{err: errors.New("miss in " + r.Name)}
		}
		_, err = ProxyWrapperWithChecks(ctx, rpfRegistries(), nil, firewall, fail, rpfInfo(), false, true)
		require.ErrorIs(t, err, usererror.ErrArtifactBlocked)
		require.Equal(t, 1, firewall.calls)

		firewall = &rpfFirewall{err: errors.New("firewall down")}
		_, err = ProxyWrapperWithChecks(ctx, rpfRegistries(), nil, firewall, fail, rpfInfo(), false, true)
		require.ErrorContains(t, err, "firewall down")

		firewall = &rpfFirewall{}
		_, err = ProxyWrapperWithChecks(ctx, rpfRegistries(), nil, firewall, fail, rpfInfo(), false, true)
		require.Error(t, err)
		require.Equal(t, 2, firewall.calls)
	})

	t.Run("no proxy variant never reaches upstream registries", func(t *testing.T) {
		firewall := &rpfFirewall{}
		finder := &rpfQuarantineFinder{}
		fail := func(r registrytypes.Registry, _ pkg.Artifact) response.Response {
			return &rpfResponse{err: errors.New("miss in " + r.Name)}
		}
		_, err := NoProxyWrapperWithChecks(ctx, rpfRegistries(), finder, firewall, fail, rpfInfo(), true, true)
		require.ErrorContains(t, err, "miss in virtual")
		require.Zero(t, firewall.calls)
		require.Len(t, finder.gotTypes, 1)
	})
}

func TestRpfSearchPackagesProxyWrapper(t *testing.T) {
	ctx := context.Background()
	rpfRegisterStubs(t)

	hitsByRegistry := map[string][]any{
		"virtual": {"v1", "v2"},
		"up-a":    {"a1", "a2", "a3"},
		"up-b":    {"b1"},
	}
	type call struct {
		registry      string
		limit, offset int
	}
	var calls []call
	search := func(r registrytypes.Registry, _ pkg.Artifact, limit, offset int) response.Response {
		calls = append(calls, call{r.Name, limit, offset})
		if r.Name == "up-a" {
			return &rpfResponse{err: errors.New("upstream down")}
		}
		return &rpfResponse{hits: hitsByRegistry[r.Name]}
	}
	extract := func(r response.Response) ([]any, int64) {
		rr, ok := r.(*rpfResponse)
		if !ok {
			return nil, 0
		}
		return rr.hits, int64(len(rr.hits))
	}

	t.Run("aggregates until the limit is reached", func(t *testing.T) {
		calls = nil
		results, total, err := SearchPackagesProxyWrapper(ctx, rpfRegistries(), search, extract, rpfInfo(), 3, 0)
		require.NoError(t, err)
		require.Equal(t, []any{"v1", "v2", "b1"}, results)
		require.Equal(t, int64(3), total)
		require.Equal(t, []call{{"virtual", 3, 0}, {"up-a", 1, 0}, {"up-b", 1, 0}}, calls)
	})

	t.Run("offset is consumed by earlier registries", func(t *testing.T) {
		calls = nil
		_, _, err := SearchPackagesProxyWrapper(ctx, rpfRegistries(), search, extract, rpfInfo(), 10, 5)
		require.NoError(t, err)
		require.Equal(t, []call{{"virtual", 10, 5}, {"up-a", 8, 3}, {"up-b", 8, 3}}, calls)
	})

	t.Run("a satisfied limit stops the search", func(t *testing.T) {
		calls = nil
		results, _, err := SearchPackagesProxyWrapper(ctx, rpfRegistries(), search, extract, rpfInfo(), 2, 0)
		require.NoError(t, err)
		require.Equal(t, []any{"v1", "v2"}, results)
		require.Len(t, calls, 1)

		calls = nil
		results, total, err := SearchPackagesProxyWrapper(ctx, rpfRegistries(), search, extract, rpfInfo(), 0, 0)
		require.NoError(t, err)
		require.Empty(t, results)
		require.Zero(t, total)
		require.Empty(t, calls)
	})

	t.Run("lookup failure and skipped registries", func(t *testing.T) {
		_, _, err := SearchPackagesProxyWrapper(ctx, rpfRegistries(), search, extract, rpfInfoFor("missing"), 5, 0)
		require.Error(t, err)

		results, _, err := SearchPackagesProxyWrapper(ctx, rpfRegistries(), search, extract, rpfInfoFor("blocked"), 5, 0)
		require.NoError(t, err)
		require.Empty(t, results)
	})

	t.Run("registries without an implementation are ignored", func(t *testing.T) {
		saved := TypeRegistry
		TypeRegistry = map[string]pkg.Artifact{}
		defer func() { TypeRegistry = saved }()
		results, _, err := SearchPackagesProxyWrapper(ctx, rpfRegistries(), search, extract, rpfInfo(), 5, 0)
		require.NoError(t, err)
		require.Empty(t, results)
	})
}

func TestRpfGetCompletePath(t *testing.T) {
	require.Equal(t, "demo/1.0/file.bin", GetCompletePath(rpfInfo(), "file.bin"))
	empty := commonstype.ArtifactInfo{}
	require.Equal(t, "file.bin", GetCompletePath(empty, "file.bin"))
}
