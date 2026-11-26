//  Copyright 2026 Harness, Inc.
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

package metadata

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	gitnessstore "github.com/harness/gitness/store"

	"github.com/stretchr/testify/require"
)

// rmdAuthFailureCase describes a scenario that diverges from the happy path.
type rmdFailureCase struct {
	name   string
	ctx    context.Context
	mutate func(*APIController)
	want   any
}

func rmdRunFailureCases[Req any, Resp any](
	t *testing.T, call func(c *APIController, ctx context.Context, r Req) (Resp, error), req Req,
	cases []rmdFailureCase,
) {
	t.Helper()
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			c := rmdNewCtrl(tt.mutate)
			ctx := tt.ctx
			if ctx == nil {
				ctx = rmdAdminCtx()
			}
			resp, err := call(c, ctx, req)
			require.NoError(t, err)
			require.IsType(t, tt.want, resp)
		})
	}
}

func rmdVirtualRepoEntity() *types.Registry {
	return &types.Registry{
		ID: 5, UUID: "uuid", Name: "reg", ParentID: 2, Type: artifact.RegistryTypeVIRTUAL,
		PackageType: artifact.PackageTypeNPM, UpstreamProxies: []int64{9}, CreatedAt: time.UnixMilli(1),
		UpdatedAt: time.UnixMilli(2),
	}
}

func rmdUpstreamEntity() *types.UpstreamProxy {
	return &types.UpstreamProxy{
		ID: 3, RegistryID: 5, RepoKey: "reg", RepoURL: "https://up.example.com", Source: "Custom",
		RepoAuthType: string(artifact.AuthTypeAnonymous), PackageType: artifact.PackageTypeNPM,
		CreatedAt: time.UnixMilli(1), UpdatedAt: time.UnixMilli(2),
	}
}

func TestRmdGetRegistry(t *testing.T) {
	req := artifact.GetRegistryRequestObject{RegistryRef: "root/reg"}
	call := func(c *APIController, ctx context.Context, r artifact.GetRegistryRequestObject) (
		artifact.GetRegistryResponseObject, error,
	) {
		return c.GetRegistry(ctx, r)
	}
	virtual := func(c *APIController) {
		c.RegistryRepository = &rmdRegRepo{byParentName: rmdVirtualRepoEntity(), upstreamKeys: []string{"up"}}
	}
	upstream := func(c *APIController) {
		c.RegistryRepository = &rmdRegRepo{byParentName: &types.Registry{ID: 5, Name: "reg"}}
		c.UpstreamProxyStore = rmdUpstreamRepo{byIdent: rmdUpstreamEntity()}
	}

	rmdRunFailureCases(t, call, req, []rmdFailureCase{
		{
			name: "base_info_not_found",
			mutate: func(c *APIController) {
				c.RegistryMetadataHelper = rmdHelper{infoErr: fmt.Errorf("wrapped: %w", gitnessstore.ErrResourceNotFound)}
			},
			want: artifact.GetRegistry404JSONResponse{},
		},
		{
			name: "base_info_other_error",
			mutate: func(c *APIController) {
				c.RegistryMetadataHelper = rmdHelper{infoErr: errors.New("boom")}
			},
			want: artifact.GetRegistry400JSONResponse{},
		},
		{
			name: "space_lookup_error",
			mutate: func(c *APIController) {
				c.SpaceFinder = rmdSpaceFinder{err: errors.New("boom")}
			},
			want: artifact.GetRegistry400JSONResponse{},
		},
		{
			name:   "anonymous_denied",
			ctx:    rmdAnonCtx(),
			mutate: func(c *APIController) { c.Authorizer = rmdAuthorizer{allow: false} },
			want:   artifact.GetRegistry401JSONResponse{},
		},
		{
			name:   "forbidden",
			mutate: func(c *APIController) { c.Authorizer = rmdAuthorizer{allow: false} },
			want:   artifact.GetRegistry403JSONResponse{},
		},
		{
			name: "registry_not_found",
			mutate: func(c *APIController) {
				c.RegistryRepository = &rmdRegRepo{byParentNameE: gitnessstore.ErrResourceNotFound}
			},
			want: artifact.GetRegistry404JSONResponse{},
		},
		{
			name: "registry_lookup_error",
			mutate: func(c *APIController) {
				c.RegistryRepository = &rmdRegRepo{byParentNameE: errors.New("db")}
			},
			want: artifact.GetRegistry500JSONResponse{},
		},
		{
			name: "virtual_cleanup_policy_error",
			mutate: func(c *APIController) {
				virtual(c)
				c.CleanupPolicyStore = rmdCleanupRepo{getErr: errors.New("db")}
			},
			want: artifact.GetRegistry500JSONResponse{},
		},
		{
			name: "virtual_without_name_is_not_found",
			mutate: func(c *APIController) {
				c.RegistryRepository = &rmdRegRepo{
					byParentName: &types.Registry{Type: artifact.RegistryTypeVIRTUAL},
				}
			},
			want: artifact.GetRegistry404JSONResponse{},
		},
		{
			name: "virtual_public_access_error",
			mutate: func(c *APIController) {
				virtual(c)
				c.PublicAccess = rmdPublicAccess{getErr: errors.New("pa")}
			},
			want: artifact.GetRegistry500JSONResponse{},
		},
		{
			name:   "virtual_success",
			mutate: virtual,
			want:   artifact.GetRegistry200JSONResponse{},
		},
		{
			name: "upstream_missing_record_is_not_found",
			mutate: func(c *APIController) {
				c.RegistryRepository = &rmdRegRepo{byParentName: &types.Registry{ID: 5, Name: "reg"}}
				c.UpstreamProxyStore = rmdUpstreamRepo{byIdent: &types.UpstreamProxy{}}
			},
			want: artifact.GetRegistry404JSONResponse{},
		},
		{
			name: "upstream_lookup_error",
			mutate: func(c *APIController) {
				upstream(c)
				c.UpstreamProxyStore = rmdUpstreamRepo{byIdent: rmdUpstreamEntity(), byIdentErr: errors.New("db")}
			},
			want: artifact.GetRegistry500JSONResponse{},
		},
		{
			name: "upstream_public_access_error",
			mutate: func(c *APIController) {
				upstream(c)
				c.PublicAccess = rmdPublicAccess{getErr: errors.New("pa")}
			},
			want: artifact.GetRegistry500JSONResponse{},
		},
		{
			name:   "upstream_success",
			mutate: upstream,
			want:   artifact.GetRegistry200JSONResponse{},
		},
	})

	t.Run("virtual_response_payload", func(t *testing.T) {
		c := rmdNewCtrl(virtual)
		resp, err := c.GetRegistry(rmdAdminCtx(), req)
		require.NoError(t, err)
		ok, isOK := resp.(artifact.GetRegistry200JSONResponse)
		require.True(t, isOK)
		require.Equal(t, "reg", ok.Data.Identifier)
		cfg, cfgErr := ok.Data.Config.AsVirtualConfig()
		require.NoError(t, cfgErr)
		require.Equal(t, []string{"up"}, *cfg.UpstreamProxies)
	})
}

func TestRmdGetAllRegistries(t *testing.T) {
	size := artifact.PageSize(10)
	req := artifact.GetAllRegistriesRequestObject{
		SpaceRef: "root", Params: artifact.GetAllRegistriesParams{Size: &size},
	}
	call := func(c *APIController, ctx context.Context, r artifact.GetAllRegistriesRequestObject) (
		artifact.GetAllRegistriesResponseObject, error,
	) {
		return c.GetAllRegistries(ctx, r)
	}
	withParams := func(mod func(p *artifact.GetAllRegistriesParams)) artifact.GetAllRegistriesRequestObject {
		r := req
		mod(&r.Params)
		return r
	}
	scopeOf := func(v string) *artifact.GetAllRegistriesParamsScope {
		s := artifact.GetAllRegistriesParamsScope(v)
		return &s
	}
	metas := &[]store.RegistryMetadata{
		{RegUUID: "u1", RegIdentifier: "r1", ParentID: 2, PackageType: artifact.PackageTypeNPM,
			Type: artifact.RegistryTypeVIRTUAL, LastModified: time.UnixMilli(5), Labels: []string{"l"},
			Description: "d", ArtifactCount: 3, DownloadCount: 4, Size: 2048},
		{RegUUID: "u2", RegIdentifier: "r2", ParentID: 0, PackageType: artifact.PackageTypeGENERIC,
			Type: artifact.RegistryTypeUPSTREAM, DeletedAt: func() *time.Time { d := time.UnixMilli(7); return &d }()},
	}
	happy := func(c *APIController) {
		c.RegistryRepository = &rmdRegRepo{all: metas, count: 2}
	}

	rmdRunFailureCases(t, call, req, []rmdFailureCase{
		{
			name:   "base_info_error",
			mutate: func(c *APIController) { c.RegistryMetadataHelper = rmdHelper{infoErr: errors.New("x")} },
			want:   artifact.GetAllRegistries400JSONResponse{},
		},
		{
			name:   "space_error",
			mutate: func(c *APIController) { c.SpaceFinder = rmdSpaceFinder{err: errors.New("x")} },
			want:   artifact.GetAllRegistries400JSONResponse{},
		},
		{
			name:   "forbidden",
			mutate: func(c *APIController) { c.Authorizer = rmdAuthorizer{allow: false} },
			want:   artifact.GetAllRegistries403JSONResponse{},
		},
		{
			name:   "invalid_package_type",
			mutate: func(c *APIController) { c.PackageWrapper = rmdPkgWrapper{invalidPkgType: true} },
			want:   artifact.GetAllRegistries400JSONResponse{},
		},
		{
			name:   "invalid_repo_type",
			mutate: func(c *APIController) { c.PackageWrapper = rmdPkgWrapper{invalidRepoType: true} },
			want:   artifact.GetAllRegistries400JSONResponse{},
		},
		{
			name: "store_error",
			mutate: func(c *APIController) {
				c.RegistryRepository = &rmdRegRepo{allErr: errors.New("db")}
			},
			want: artifact.GetAllRegistries500JSONResponse{},
		},
		{name: "success", mutate: happy, want: artifact.GetAllRegistries200JSONResponse{}},
	})

	t.Run("negative_page_size_is_rejected", func(t *testing.T) {
		neg := artifact.PageSize(-1)
		page := artifact.PageNumber(1)
		resp, err := rmdNewCtrl(happy).GetAllRegistries(rmdAdminCtx(), withParams(func(p *artifact.GetAllRegistriesParams) {
			p.Size = &neg
			p.Page = &page
		}))
		require.NoError(t, err)
		require.IsType(t, artifact.GetAllRegistries400JSONResponse{}, resp)
	})

	t.Run("invalid_scope", func(t *testing.T) {
		resp, err := rmdNewCtrl(happy).GetAllRegistries(rmdAdminCtx(), withParams(func(p *artifact.GetAllRegistriesParams) {
			p.Scope = scopeOf("sideways")
		}))
		require.NoError(t, err)
		require.IsType(t, artifact.GetAllRegistries400JSONResponse{}, resp)
	})

	t.Run("ancestors_scope", func(t *testing.T) {
		r := withParams(func(p *artifact.GetAllRegistriesParams) { p.Scope = scopeOf("ancestors") })
		ok := rmdNewCtrl(happy, func(c *APIController) { c.SpaceStore = rmdSpaceStore{ancestors: []int64{1, 2}} })
		resp, err := ok.GetAllRegistries(rmdAdminCtx(), r)
		require.NoError(t, err)
		require.IsType(t, artifact.GetAllRegistries200JSONResponse{}, resp)

		bad := rmdNewCtrl(happy, func(c *APIController) { c.SpaceStore = rmdSpaceStore{err: errors.New("db")} })
		resp, err = bad.GetAllRegistries(rmdAdminCtx(), r)
		require.NoError(t, err)
		require.IsType(t, artifact.GetAllRegistries500JSONResponse{}, resp)
	})

	t.Run("recursive_flag_uses_ancestors", func(t *testing.T) {
		rec := artifact.RecursiveParam(true)
		r := withParams(func(p *artifact.GetAllRegistriesParams) { p.Recursive = &rec })
		bad := rmdNewCtrl(happy, func(c *APIController) { c.SpaceStore = rmdSpaceStore{err: errors.New("db")} })
		resp, err := bad.GetAllRegistries(rmdAdminCtx(), r)
		require.NoError(t, err)
		require.IsType(t, artifact.GetAllRegistries500JSONResponse{}, resp)
	})

	t.Run("descendants_scope", func(t *testing.T) {
		r := withParams(func(p *artifact.GetAllRegistriesParams) { p.Scope = scopeOf("descendants") })
		ok := rmdNewCtrl(happy, func(c *APIController) { c.SpaceStore = rmdSpaceStore{descendants: []int64{3}} })
		resp, err := ok.GetAllRegistries(rmdAdminCtx(), r)
		require.NoError(t, err)
		require.IsType(t, artifact.GetAllRegistries200JSONResponse{}, resp)

		bad := rmdNewCtrl(happy, func(c *APIController) { c.SpaceStore = rmdSpaceStore{descErr: errors.New("db")} })
		resp, err = bad.GetAllRegistries(rmdAdminCtx(), r)
		require.NoError(t, err)
		require.IsType(t, artifact.GetAllRegistries500JSONResponse{}, resp)
	})

	t.Run("response_mapping", func(t *testing.T) {
		resp, err := rmdNewCtrl(happy).GetAllRegistries(rmdAdminCtx(), req)
		require.NoError(t, err)
		ok, isOK := resp.(artifact.GetAllRegistries200JSONResponse)
		require.True(t, isOK)
		list := ok.Data.Registries
		require.Len(t, list, 2)
		require.Equal(t, "root/r1", *list[0].Path)
		require.Equal(t, rmdRegistryBase+"/root/r1", list[0].Url)
		require.Equal(t, int64(3), *list[0].ArtifactsCount)
		require.Equal(t, int64(4), *list[0].DownloadsCount)
		require.Equal(t, []string{"l"}, *list[0].Labels)
		require.Equal(t, "d", *list[0].Description)
		require.Equal(t, GetSize(2048), *list[0].RegistrySize)
		require.Nil(t, list[0].DeletedAt)

		require.Equal(t, rmdRegistryBase+"/root/generic/r2", list[1].Url)
		require.Empty(t, *list[1].Path, "registries without a parent have no path")
		require.Nil(t, list[1].ArtifactsCount)
		require.Nil(t, list[1].DownloadsCount)
		require.Nil(t, list[1].Labels)
		require.Equal(t, "7", *list[1].DeletedAt)
	})
}

func TestRmdGetRegistryPath(t *testing.T) {
	ctx := context.Background()
	c := rmdNewCtrl()
	require.Equal(t, "root/reg", c.GetRegistryPath(ctx, 2, "reg"))
	require.Empty(t, c.GetRegistryPath(ctx, 0, "reg"))
	bad := rmdNewCtrl(func(c *APIController) { c.SpaceFinder = rmdSpaceFinder{err: errors.New("x")} })
	require.Empty(t, bad.GetRegistryPath(ctx, 2, "reg"))
	missing := rmdNewCtrl(func(c *APIController) { c.SpaceFinder = rmdSpaceFinder{} })
	require.Empty(t, missing.GetRegistryPath(ctx, 2, "reg"))
}

func TestRmdDeleteRegistry(t *testing.T) {
	req := artifact.DeleteRegistryRequestObject{RegistryRef: "root/reg"}
	call := func(c *APIController, ctx context.Context, r artifact.DeleteRegistryRequestObject) (
		artifact.DeleteRegistryResponseObject, error,
	) {
		return c.DeleteRegistry(ctx, r)
	}
	found := func(c *APIController) {
		c.RegistryRepository = &rmdRegRepo{byParentName: rmdVirtualRepoEntity()}
	}

	rmdRunFailureCases(t, call, req, []rmdFailureCase{
		{
			name:   "base_info_error",
			mutate: func(c *APIController) { c.RegistryMetadataHelper = rmdHelper{infoErr: errors.New("x")} },
			want:   artifact.DeleteRegistry400JSONResponse{},
		},
		{
			name:   "space_error",
			mutate: func(c *APIController) { c.SpaceFinder = rmdSpaceFinder{err: errors.New("x")} },
			want:   artifact.DeleteRegistry400JSONResponse{},
		},
		{
			name:   "anonymous_denied",
			ctx:    rmdAnonCtx(),
			mutate: func(c *APIController) { c.Authorizer = rmdAuthorizer{allow: false} },
			want:   artifact.DeleteRegistry401JSONResponse{},
		},
		{
			name:   "forbidden",
			mutate: func(c *APIController) { c.Authorizer = rmdAuthorizer{allow: false} },
			want:   artifact.DeleteRegistry403JSONResponse{},
		},
		{
			name: "not_found",
			mutate: func(c *APIController) {
				c.RegistryRepository = &rmdRegRepo{byParentNameE: gitnessstore.ErrResourceNotFound}
			},
			want: artifact.DeleteRegistry404JSONResponse{},
		},
		{
			name: "lookup_error",
			mutate: func(c *APIController) {
				c.RegistryRepository = &rmdRegRepo{byParentNameE: errors.New("db")}
			},
			want: artifact.DeleteRegistry500JSONResponse{},
		},
		{
			name: "used_as_upstream_lookup_error",
			mutate: func(c *APIController) {
				c.RegistryRepository = &rmdRegRepo{byParentName: rmdVirtualRepoEntity(), usedByErr: errors.New("db")}
			},
			want: artifact.DeleteRegistry400JSONResponse{},
		},
		{
			name: "used_as_upstream",
			mutate: func(c *APIController) {
				c.RegistryRepository = &rmdRegRepo{
					byParentName: rmdVirtualRepoEntity(), usedByIDs: []int64{11},
					byIDIn: &[]types.Registry{{Name: "user-reg", ParentID: 2}},
				}
			},
			want: artifact.DeleteRegistry400JSONResponse{},
		},
		{
			name: "used_as_upstream_registries_lookup_error",
			mutate: func(c *APIController) {
				c.RegistryRepository = &rmdRegRepo{
					byParentName: rmdVirtualRepoEntity(), usedByIDs: []int64{11}, byIDInErr: errors.New("db"),
				}
			},
			want: artifact.DeleteRegistry400JSONResponse{},
		},
		{
			name: "public_access_delete_error",
			mutate: func(c *APIController) {
				found(c)
				c.PublicAccess = rmdPublicAccess{deleteErr: errors.New("pa")}
			},
			want: artifact.DeleteRegistry500JSONResponse{},
		},
		{
			name: "registry_finder_delete_error",
			mutate: func(c *APIController) {
				found(c)
				c.RegFinder = rmdRegFinder{deleteErr: errors.New("delete query failed: details")}
			},
			want: artifact.DeleteRegistry500JSONResponse{},
		},
		{name: "success_virtual", mutate: found, want: artifact.DeleteRegistry200JSONResponse{}},
		{
			name: "success_upstream_with_failing_audit",
			mutate: func(c *APIController) {
				up := rmdVirtualRepoEntity()
				up.Type = artifact.RegistryTypeUPSTREAM
				c.RegistryRepository = &rmdRegRepo{byParentName: up}
				c.AuditService = rmdAudit{err: errors.New("audit down")}
			},
			want: artifact.DeleteRegistry200JSONResponse{},
		},
	})

	t.Run("delete_error_message_is_masked", func(t *testing.T) {
		c := rmdNewCtrl(found, func(c *APIController) {
			c.RegFinder = rmdRegFinder{deleteErr: errors.New("delete query failed: secret sql")}
		})
		resp, err := c.DeleteRegistry(rmdAdminCtx(), req)
		require.NoError(t, err)
		e, ok := resp.(artifact.DeleteRegistry500JSONResponse)
		require.True(t, ok)
		require.NotContains(t, e.Message, "secret sql")
	})

	t.Run("used_as_upstream_message_lists_users", func(t *testing.T) {
		c := rmdNewCtrl(func(c *APIController) {
			c.RegistryRepository = &rmdRegRepo{
				byParentName: rmdVirtualRepoEntity(), usedByIDs: []int64{11},
				byIDIn: &[]types.Registry{{Name: "user-reg", ParentID: 2}},
			}
		})
		resp, err := c.DeleteRegistry(rmdAdminCtx(), req)
		require.NoError(t, err)
		e, ok := resp.(artifact.DeleteRegistry400JSONResponse)
		require.True(t, ok)
		require.Contains(t, e.Message, "user-reg (root)")
	})
}
